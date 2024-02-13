use bellpepper::gadgets::{multipack::pack_bits, sha256::sha256};
use bellpepper_core::{boolean::Boolean, ConstraintSystem, SynthesisError};
use lurk_macros::Coproc;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::marker::PhantomData;

use lurk::{
    circuit::gadgets::pointer::AllocatedPtr,
    coprocessor::{CoCircuit, Coprocessor},
    field::LurkField,
    lem::{pointers::Ptr, store::Store},
};

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Sha256Coprocessor<F: LurkField> {
    n: usize,
    pub(crate) _p: PhantomData<F>,
}

fn synthesize_sha256<F: LurkField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    ptrs: &[AllocatedPtr<F>],
) -> Result<AllocatedPtr<F>, SynthesisError> {
    let zero = Boolean::constant(false);

    let mut bits = vec![];

    let hash_bits = ptrs[0]
        .hash()
        .to_bits_le_strict(&mut cs.namespace(|| "preimage_hash_bits"))?;

    let pad_to_next_len_multiple_of_8 = |bits: &mut Vec<_>| {
        bits.resize((bits.len() + 7) / 8 * 8, zero.clone());
    };

    bits.extend(hash_bits.clone());
    pad_to_next_len_multiple_of_8(&mut bits); // need 256 bits (or some multiple of 8).

    bits.reverse();

    let mut digest_bits = sha256(cs.namespace(|| "digest_bits"), &bits)?;

    digest_bits.reverse();

    // Fine to lose the last <1 bit of precision.
    let digest_scalar = pack_bits(cs.namespace(|| "digest_scalar"), &digest_bits)?;
    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        F::from_u64(4),
        digest_scalar,
    )
}

fn compute_sha256<F: LurkField>(s: &Store<F>, ptrs: &[Ptr]) -> F {
    let z_ptrs = ptrs.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();
    let mut hasher = Sha256::new();

    let mut input = vec![0u8; 32];

    let hash_zptr = z_ptrs[0].value();
    input[0..32].copy_from_slice(&hash_zptr.to_bytes());

    input.reverse();

    let hex_string: String = input.iter().rev().map(|byte| format!("{:02x}", byte)).collect();
    println!("r {}", hex_string);

    hasher.update(input);
    let mut bytes = hasher.finalize();
    bytes.reverse();
    let l = bytes.len();
    // Discard the two most significant bits.
    bytes[l - 1] &= 0b00111111;

    F::from_bytes(&bytes).unwrap()
}

impl<F: LurkField> CoCircuit<F> for Sha256Coprocessor<F> {
    fn arity(&self) -> usize {
        self.n
    }

    #[inline]
    fn synthesize_simple<CS: ConstraintSystem<F>>(
        &self,
        cs: &mut CS,
        _g: &lurk::lem::circuit::GlobalAllocator<F>,
        _s: &lurk::lem::store::Store<F>,
        _not_dummy: &Boolean,
        args: &[AllocatedPtr<F>],
    ) -> Result<AllocatedPtr<F>, SynthesisError> {
        synthesize_sha256(cs, args)
    }
}

impl<F: LurkField> Coprocessor<F> for Sha256Coprocessor<F> {
    fn eval_arity(&self) -> usize {
        self.n
    }

    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        s.num(compute_sha256(s, args))
    }
}

impl<F: LurkField> Sha256Coprocessor<F> {
    pub fn new() -> Self {
        Self {
            n: 1,
            _p: Default::default(),
        }
    }
}

#[derive(Clone, Debug, Coproc, Serialize, Deserialize)]
pub enum Sha256Coproc<F: LurkField> {
    SC(Sha256Coprocessor<F>),
}
