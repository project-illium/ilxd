use bellpepper::gadgets::{multipack::pack_bits, blake2s::blake2s};
use bellpepper_core::{boolean::Boolean, ConstraintSystem, SynthesisError};
use lurk_macros::Coproc;
use serde::{Deserialize, Serialize};
use blake2s_simd::Params as Blake2sParams;
use std::marker::PhantomData;

use lurk::{
    circuit::gadgets::pointer::AllocatedPtr,
    field::LurkField,
    lem::{pointers::Ptr, store::Store},
    coprocessor::{CoCircuit, Coprocessor},
};

/*
(letrec ((cat-and-hash (lambda (a b)
            (eval (cons '.lurk.blake2s (cons a (cons b nil)))))))
  (cat-and-hash 0x18b1a7da2e8bc9c7633224d4df95f5730e521bbc6d57c8db9ab51a2f963d703b 500))
 */

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Blake2sCoprocessor<F: LurkField> {
    n: usize,
    pub(crate) _p: PhantomData<F>,
}

fn synthesize_blake2s<F: LurkField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    ptrs: &[AllocatedPtr<F>],
) -> Result<AllocatedPtr<F>, SynthesisError> {
    let zero = Boolean::constant(false);
    let personalization: [u8; 8] = [0; 8];

    let mut bits = vec![];

    let pad_to_next_len_multiple_of_8 = |bits: &mut Vec<_>| {
        bits.resize((bits.len() + 7) / 8 * 8, zero.clone());
    };

    for ptr in ptrs.iter().rev() {
        let hash_bits = ptr
            .hash()
            .to_bits_le_strict(&mut cs.namespace(|| "preimage_hash_bits"))?;

        bits.extend(hash_bits);
        pad_to_next_len_multiple_of_8(&mut bits); // need 256 bits (or some multiple of 8).
    }

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

    // Fine to lose the last <1 bit of precision.
    let digest_scalar = pack_bits(cs.namespace(|| "digest_scalar"), &little_endian_digest_bits)?;

    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        F::from(4),
        digest_scalar,
    )
}

fn compute_blake2s<F: LurkField>(s: &Store<F>, ptrs: &[Ptr]) -> F {
    let z_ptrs = ptrs.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();

    let personalization: [u8; 8] = [0; 8];
    let mut hasher = Blake2sParams::new()
        .hash_length(32)
        .personal(&personalization)
        .to_state();

    let mut input = vec![0u8; 64];

    for (i, z_ptr) in z_ptrs.iter().rev().enumerate() {
        let hash_zptr = z_ptr.value();

        let start = 32 * i;
        let end = start + 32;

        input[start..end].copy_from_slice(&hash_zptr.to_bytes());
    }

    input.reverse();

    hasher.update(&input);
    let mut bytes = hasher.finalize().as_bytes().to_vec();
    bytes.reverse();
    let l = bytes.len();
    // Discard the two most significant bits.
    bytes[l - 1] &= 0b00111111;

    F::from_bytes(&bytes).unwrap()
}

impl<F: LurkField> CoCircuit<F> for Blake2sCoprocessor<F> {
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
        synthesize_blake2s(cs, args)
    }
}

impl<F: LurkField> Coprocessor<F> for Blake2sCoprocessor<F> {
    fn eval_arity(&self) -> usize {
        self.n
    }

    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        s.num(compute_blake2s(s, &args))
    }
}

impl<F: LurkField> Blake2sCoprocessor<F> {
    pub fn new() -> Self {
        Self {
            n: 2,
            _p: Default::default(),
        }
    }
}

#[derive(Clone, Debug, Coproc, Serialize, Deserialize)]
pub enum Blake2sCoproc<F: LurkField> {
    SC(Blake2sCoprocessor<F>),
}