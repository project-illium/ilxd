use std::hash::Hash;
use bellpepper::gadgets::{multipack::pack_bits};
use bellpepper_core::{boolean::Boolean, ConstraintSystem, SynthesisError};
use lurk_macros::Coproc;
use serde::{Deserialize, Serialize, Serializer};
use std::marker::PhantomData;
use bellpepper_core::num::AllocatedNum;

use crate::{
    self as lurk,
    circuit::gadgets::pointer::AllocatedPtr,
    field::LurkField,
    lem::{pointers::Ptr, store::Store, multiframe::MultiFrame},
    tag::{ExprTag, Tag},
    z_ptr::ZPtr,
};

use super::{CoCircuit, Coprocessor, gadgets};

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct XorCoprocessor<F: LurkField> {
    n: usize,
    pub(crate) _p: PhantomData<F>,
}

fn synthesize_xor<F: LurkField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    g: &lurk::lem::circuit::GlobalAllocator<F>,
    s: &lurk::lem::store::Store<F>,
    ptrs: &[AllocatedPtr<F>],
) -> Result<AllocatedPtr<F>, SynthesisError> {
    let zero = Boolean::constant(false);

    let mut hash_a = ptrs[0]
        .hash()
        .to_bits_le_strict(&mut cs.namespace(|| "hash_a_bits"))?;
    hash_a.push(zero.clone()); // need 256 bits (or some multiple of 8).

    let mut hash_b = ptrs[1]
        .hash()
        .to_bits_le_strict(&mut cs.namespace(|| "hash_b_bits"))?;
    hash_b.push(zero.clone()); // need 256 bits (or some multiple of 8).

    let mut xor_result = Vec::new();
    for i in 0..hash_a.len() {
        let xor_element: Boolean = Boolean::xor(
            &mut cs.namespace(|| format!("xor_{}", i)),
            hash_a.get(i).unwrap(),
            hash_b.get(i).unwrap(),
        )?;
        xor_result.push(xor_element);
    }

    let x_scalar = pack_bits(cs.namespace(|| "x_scalar"), &xor_result)?;
    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        ExprTag::Num.to_field(),
        x_scalar,
    )
}

fn compute_xor<F: LurkField, T: Tag>(s: &Store<F>, z_ptrs: &[ZPtr<T, F>]) -> Ptr {

    let hash_a = z_ptrs[0].value().to_bytes();
    let hash_b = z_ptrs[1].value().to_bytes();

    let mut xor_result: Vec<u8> = hash_a.iter()
        .zip(hash_b.iter())
        .map(|(&x, &y)| x ^ y)
        .collect();

    let l = xor_result.len();

    xor_result[l - 1] &= 0b00111111;
    s.num(F::from_bytes(&xor_result).unwrap())
}

impl<F: LurkField> CoCircuit<F> for XorCoprocessor<F> {
    fn arity(&self) -> usize {
        self.n
    }

    #[inline]
    fn synthesize_simple<CS: ConstraintSystem<F>>(
        &self,
        cs: &mut CS,
        g: &lurk::lem::circuit::GlobalAllocator<F>,
        s: &lurk::lem::store::Store<F>,
        _not_dummy: &Boolean,
        args: &[AllocatedPtr<F>],
    ) -> Result<AllocatedPtr<F>, SynthesisError> {
        synthesize_xor(cs, g, s, args)
    }
}

impl<F: LurkField> Coprocessor<F> for XorCoprocessor<F> {
    fn eval_arity(&self) -> usize {
        self.n
    }

    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        let z_ptrs = args.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();
        compute_xor(s, &z_ptrs)
    }
}

impl<F: LurkField> XorCoprocessor<F> {
    pub fn new() -> Self {
        Self {
            n: 2,
            _p: Default::default(),
        }
    }
}

#[derive(Clone, Debug, Coproc, Serialize, Deserialize)]
pub enum XorCoproc<F: LurkField> {
    SC(XorCoprocessor<F>),
}

/// Retrieves the `Ptr` that corresponds to `a_ptr` by using the `Store` as the
/// hint provider
#[allow(dead_code)]
fn get_ptr<F: LurkField>(a_ptr: &AllocatedPtr<F>, store: &Store<F>) -> Result<Ptr, SynthesisError> {
    let z_ptr = crate::lem::pointers::ZPtr::from_parts(
        Tag::from_field(
            &a_ptr
                .tag()
                .get_value()
                .ok_or_else(|| SynthesisError::AssignmentMissing)?,
        )
            .ok_or_else(|| SynthesisError::Unsatisfiable)?,
        a_ptr
            .hash()
            .get_value()
            .ok_or_else(|| SynthesisError::AssignmentMissing)?,
    );
    Ok(store.to_ptr(&z_ptr))
}

impl<'a, Fq> Serialize for MultiFrame<'a, Fq, XorCoproc<Fq>> where Fq: Serialize + LurkField {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
    {
        // This is a dummy implementation that does nothing
        serializer.serialize_unit()
    }
}