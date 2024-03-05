use bellpepper::gadgets::{multipack::pack_bits, blake2s::blake2s};
use bellpepper_core::{boolean::Boolean, ConstraintSystem, SynthesisError, num::AllocatedNum};
use lurk_macros::Coproc;
use serde::{Deserialize, Serialize};
use blake2s_simd::Params as Blake2sParams;
use std::marker::PhantomData;
use lazy_static::lazy_static;

use lurk::{
    circuit::gadgets::pointer::AllocatedPtr,
    field::LurkField,
    lem::{pointers::Ptr, store::Store},
    coprocessor::{CoCircuit, Coprocessor, gadgets::chain_car_cdr},
};
use lurk::circuit::gadgets::constraints::alloc_equal;
use crate::coprocessors::utils::{pick, pick_const, open};

lazy_static! {
    static ref IO_TRUE_HASH: Vec<u8> = hex::decode("4c8ca192c0f6acba0d6816ce095040633a3ef6cb9bcea4f2b834514035f05c1f").unwrap();
    static ref IO_FALSE_HASH: Vec<u8> = hex::decode("032f240cbb095bf8e5a50533e6f86f3695048af3b7279e067364da67c2c8551d").unwrap();
}


#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MerkleCoprocessor<F: LurkField> {
    n: usize,
    pub(crate) _p: PhantomData<F>,
}

fn synthesize_merkle<F: LurkField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    g: &lurk::lem::circuit::GlobalAllocator<F>,
    s: &Store<F>,
    not_dummy: &Boolean,
    ptrs: &[AllocatedPtr<F>],
) -> Result<AllocatedPtr<F>, SynthesisError> {
    let commitment = ptrs[0].hash();
    let index = ptrs[1].hash();
    let flags = ptrs[2].hash().to_bits_le(cs.namespace(|| "flag bits"))?;
    let commit = ptrs[3].clone();
    let root = ptrs[4].hash();

    let mut hashes = open(cs, s, not_dummy, &commit)?;

    let t_tag = F::from_u64(2);
    let f_tag = F::from_u64(0);

    let zero = AllocatedNum::alloc(cs.namespace(|| "zero"), || {
        Ok(f_tag)
    })?;

    let t = AllocatedNum::alloc(cs.namespace(|| "t"), || {
        Ok(F::from_bytes(&IO_TRUE_HASH).unwrap())
    })?;
    let f = AllocatedNum::alloc(cs.namespace(|| "f"), || {
        Ok(F::from_bytes(&IO_FALSE_HASH).unwrap())
    })?;

    let computed = synthesize_cat_and_hash(&mut cs.namespace(|| "genesis"), commitment, index)?;
    let mut computed2 = computed.clone();

    // If hashes is nil then this is the genesis block and we only need to compare hash(commit || index)
    // to the root
    let nil_hashes_tag = alloc_equal(&mut cs.namespace(|| "hashes tag is nil"), hashes.tag(), &zero)?;
    let nil_hashes_value = alloc_equal(&mut cs.namespace(|| "hashes value is nil"), hashes.hash(), &f)?;
    let nil_hashes: Boolean = Boolean::and(
        &mut cs.namespace(|| "nil hashes and"),
        &nil_hashes_tag,
        &nil_hashes_value,
    )?;
    let equal_genesis_root = alloc_equal(&mut cs.namespace(|| "genesis root equal"), &computed, root)?;

    // Otherwise we need to iterate over the hashes and compute root
    let mut i = 0;
    loop {
        let (car, cdr, length) = chain_car_cdr(&mut cs.namespace(|| format!("car_cdr_{i}")), g, s, not_dummy, &hashes, 1)?;
        let end = alloc_equal(&mut cs.namespace(|| format!("end of list {i}")), &length, &zero)?;
        
        if let Some(true) | None = end.get_value() {
            break;
        }

        let left = synthesize_cat_and_hash(&mut cs.namespace(|| format!("left {i}")), &computed2, car[0].hash())?;
        let right = synthesize_cat_and_hash(&mut cs.namespace(|| format!("right {i}")), car[0].hash(), &computed2)?;

        computed2 = pick(cs.namespace(|| format!("new computed2 {i}")), &flags[i], &right, &left)?;

        hashes = cdr;

        i += 1;
    }

    let equal_computed = alloc_equal(&mut cs.namespace(|| "computed root equal"), &computed2, root)?;

    // Now we can pick which output to use
    let valid_genesis: Boolean = Boolean::and(
        &mut cs.namespace(|| "valid genesis"),
        &nil_hashes,
        &equal_genesis_root,
    )?;
    let valid_computed: Boolean = Boolean::and(
        &mut cs.namespace(|| "valid computed"),
        &nil_hashes.not(),
        &equal_computed,
    )?;
    let valid: Boolean = Boolean::or(
        &mut cs.namespace(|| "valid"),
        &valid_genesis,
        &valid_computed,
    )?;

    let resp = pick(cs.namespace(|| "pick"), &valid, &t, &f)?;

    let tag = pick_const(cs.namespace(|| "pick_tag"), &valid, t_tag, f_tag)?;

    let tag_type = tag.get_value().unwrap_or(t_tag);

    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        tag_type,
        resp,
    )
}

fn synthesize_cat_and_hash<F: LurkField, CS: ConstraintSystem<F>>(cs: &mut CS, a: &AllocatedNum<F>, b: &AllocatedNum<F>) -> Result<AllocatedNum<F>, SynthesisError> {
    let zero = Boolean::constant(false);
    let personalization: [u8; 8] = [0; 8];

    let a_bits = a.to_bits_le_strict(&mut cs.namespace(|| "preimage_a_hash_bits"))?;
    let b_bits = b.to_bits_le_strict(&mut cs.namespace(|| "preimage_b_hash_bits"))?;

    let mut bits = vec![];

    let pad_to_next_len_multiple_of_8 = |bits: &mut Vec<_>| {
        bits.resize((bits.len() + 7) / 8 * 8, zero.clone());
    };

    bits.extend(a_bits);
    pad_to_next_len_multiple_of_8(&mut bits); // need 256 bits (or some multiple of 8).

    bits.extend(b_bits);
    pad_to_next_len_multiple_of_8(&mut bits); // need 256 bits (or some multiple of 8).

    let mut little_endian_bits = Vec::new();
    let chunks = bits.chunks(8);

    // Reverse the order of chunks
    for chunk in chunks.rev() {
        little_endian_bits.extend_from_slice(chunk);
    }
    let digest_bits = blake2s(cs.namespace(|| "digest_bits"), &little_endian_bits, &personalization)?;

    let mut little_endian_digest_bits = Vec::new();
    let chunks = digest_bits.chunks(8);

    // Reverse the order of chunks
    for chunk in chunks.rev() {
        little_endian_digest_bits.extend_from_slice(chunk);
    }

    pack_bits(cs.namespace(|| "digest_scalar"), &little_endian_digest_bits)
}

fn validate_merkle_proof<F: LurkField>(s: &Store<F>, ptrs: &[Ptr]) -> Ptr {
    let z_ptrs = ptrs.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();

    let commitment = z_ptrs[0].value().to_bytes();
    let index = z_ptrs[1].value().to_bytes();
    let mut flags = z_ptrs[2].value().to_bytes();
    let commit = z_ptrs[3].value();
    let root = z_ptrs[4].value().to_bytes();

    flags.reverse();

    let mut computed = compute_cat_and_hash(commitment, index);

    if !ptrs[3].is_nil() {
        let (_, hashes) = s.open(*commit).unwrap();
        let (mut first, mut rest) = s.car_cdr(hashes).unwrap();
        let mut i = 0;
        loop {
            let first_bytes = s.hash_ptr(&first).value().to_bytes();

            // Calculate the index of the byte and the bit within that byte, starting from the end
            let byte_index = flags.len() - 1 - (i / 8);
            let bit_index = i % 8;

            // Ensure we do not go beyond the start of the flags array
            if byte_index >= flags.len() {
                break;
            }

            // Check the bit at position i in flags, starting from the LSB of the last byte
            if (flags[byte_index] >> bit_index) & 1 == 0 {
                computed = compute_cat_and_hash(computed, first_bytes);
            } else {
                computed = compute_cat_and_hash(first_bytes, computed);
            }

            if rest.is_nil() {
                break
            }

            i += 1; // Move to the next bit, going backwards through the array
            (first, rest) = s.car_cdr(&rest).unwrap();
        }
    }

    if computed == root {
        return s.intern_lurk_symbol("t");
    }
    s.intern_nil()
}

fn compute_cat_and_hash(a: Vec<u8>, b: Vec<u8>) -> Vec<u8> {
    let mut input = vec![0u8; 64];
    input[0..32].copy_from_slice(&a);
    input[32..64].copy_from_slice(&b);

    input.reverse();

    let personalization: [u8; 8] = [0; 8];
    let mut hasher = Blake2sParams::new()
        .hash_length(32)
        .personal(&personalization)
        .to_state();

    hasher.update(&input);
    let mut bytes = hasher.finalize().as_bytes().to_vec();
    bytes.reverse();
    let l = bytes.len();
    // Discard the two most significant bits.
    bytes[l - 1] &= 0b00011111;

    bytes
}

impl<F: LurkField> CoCircuit<F> for MerkleCoprocessor<F> {
    fn arity(&self) -> usize {
        self.n
    }

    #[inline]
    fn synthesize_simple<CS: ConstraintSystem<F>>(
        &self,
        cs: &mut CS,
        g: &lurk::lem::circuit::GlobalAllocator<F>,
        s: &lurk::lem::store::Store<F>,
        not_dummy: &Boolean,
        args: &[AllocatedPtr<F>],
    ) -> Result<AllocatedPtr<F>, SynthesisError> {
        synthesize_merkle(cs, g, s, not_dummy, args)
    }
}

impl<F: LurkField> Coprocessor<F> for MerkleCoprocessor<F> {
    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        validate_merkle_proof(s, &args)
    }
}

impl<F: LurkField> MerkleCoprocessor<F> {
    pub fn new() -> Self {
        Self {
            n: 5,
            _p: Default::default(),
        }
    }
}

#[derive(Clone, Debug, Coproc, Serialize, Deserialize)]
pub enum MerkleCoproc<F: LurkField> {
    SC(MerkleCoprocessor<F>),
}

#[cfg(test)]
mod tests {
    use bellpepper_core::test_cs::TestConstraintSystem;
    use halo2curves::bn256::Fr;
    use lurk::lem::circuit::GlobalAllocator;
    use super::*;

    #[test]
    fn test_merkle() {
        let store= &mut Store::<Fr>::default();

        let commitment: Fr =  bincode::deserialize(&hex::decode("4afc039dc30f50d20336caf6a26dce2d40b26ca9f69ae8afbdb53ee479f8c02b").unwrap()).unwrap();
        let root: Fr =  bincode::deserialize(&hex::decode("9391361e31556492d73813ff62379e947e95132445d09b17c3ba244ec832a40f").unwrap()).unwrap();
        let index: Fr = Fr::from_u64(10000);
        let flags: Fr = Fr::from_u64(2);

        let h0: Fr =  bincode::deserialize(&hex::decode("ffc45b9cc1f7722609c86e8fdce4f301234b4d4a2fb45cd9e935d0492f05df04").unwrap()).unwrap();
        let h1: Fr =  bincode::deserialize(&hex::decode("4fb7629dec7ec8bb7dc2bb6a549c81db0b92df07609047de9ec0be34933de41c").unwrap()).unwrap();
        let h2: Fr =  bincode::deserialize(&hex::decode("dbba427f36ba3a66bf399b8b11576737d1fd5cc15c9d6963b4c7a9ee0c76331b").unwrap()).unwrap();

        let hashes = store.cons(store.num(h0), store.cons(store.num(h1), store.cons(store.num(h2), store.intern_nil())));

        let commit = store.commit(hashes);
        let commit_zptr = store.hash_ptr(&commit);

        let args: &[Ptr] = &[
            store.num(commitment),
            store.num(index),
            store.num(flags),
            commit,
            store.num(root),
        ];

        let resp = validate_merkle_proof(store, args);
        println!("{:?}", resp);

        let mut cs = TestConstraintSystem::<Fr>::new();
        let g = &GlobalAllocator::default();

        let commitment_num = AllocatedNum::alloc(cs.namespace(|| "c num"), || {Ok(commitment)}).unwrap();
        let c = AllocatedPtr::alloc_tag(&mut cs.namespace(|| "commitment"), Fr::from_u64(4), commitment_num).unwrap();

        let index_num = AllocatedNum::alloc(cs.namespace(|| "i num"), || {Ok(index)}).unwrap();
        let i = AllocatedPtr::alloc_tag(&mut cs.namespace(|| "index"), Fr::from_u64(4), index_num).unwrap();

        let flags_num = AllocatedNum::alloc(cs.namespace(|| "f num"), || {Ok(flags)}).unwrap();
        let f = AllocatedPtr::alloc_tag(&mut cs.namespace(|| "flags"), Fr::from_u64(4), flags_num).unwrap();

        let root_num = AllocatedNum::alloc(cs.namespace(|| "r num"), || {Ok(root)}).unwrap();
        let r = AllocatedPtr::alloc_tag(&mut cs.namespace(|| "root"), Fr::from_u64(4), root_num).unwrap();

        let hashes_hash = AllocatedNum::alloc(cs.namespace(|| "h hash"), || {Ok(commit_zptr.1)}).unwrap();
        let h = AllocatedPtr::alloc_tag(&mut cs.namespace(|| "hashes"), Fr::from_u64(1), hashes_hash).unwrap();

        let ptrs: &[AllocatedPtr<Fr>] = &[
            c,
            i,
            f,
            h,
            r,
        ];
        let output = synthesize_merkle(&mut cs, g, store, &Boolean::Constant(true), ptrs).unwrap();
        assert!(cs.is_satisfied());
        println!("{:?}", output);
    }
}