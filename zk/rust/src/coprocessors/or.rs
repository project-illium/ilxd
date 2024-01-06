use bellpepper::gadgets::{multipack::pack_bits};
use bellpepper_core::{boolean::Boolean, ConstraintSystem, SynthesisError};
use lurk_macros::Coproc;
use serde::{Deserialize, Serialize};
use std::marker::PhantomData;

use lurk::{
    circuit::gadgets::pointer::AllocatedPtr,
    coprocessor::{CoCircuit, Coprocessor},
    field::LurkField,
    lem::{pointers::Ptr, store::Store},
};

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OrCoprocessor<F: LurkField> {
    n: usize,
    pub(crate) _p: PhantomData<F>,
}

fn synthesize_or<F: LurkField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    _: &lurk::lem::circuit::GlobalAllocator<F>,
    _: &lurk::lem::store::Store<F>,
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

    let mut or_result = Vec::new();
    for i in 0..hash_a.len() {
        let or_element: Boolean = Boolean::or(
            &mut cs.namespace(|| format!("or_{}", i)),
            hash_a.get(i).unwrap(),
            hash_b.get(i).unwrap(),
        )?;
        or_result.push(or_element);
    }

    let x_scalar = pack_bits(cs.namespace(|| "o_scalar"), &or_result)?;

    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        F::from(4),
        x_scalar,
    )
}

fn compute_or<F: LurkField>(s: &Store<F>, ptrs: &[Ptr]) -> Ptr {
    let z_ptrs = ptrs.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();

    let hash_a = z_ptrs[0].value().to_bytes();
    let hash_b = z_ptrs[1].value().to_bytes();

    let mut or_result: Vec<u8> = hash_a.iter()
        .zip(hash_b.iter())
        .map(|(&x, &y)| x | y)
        .collect();

    let l = or_result.len();

    or_result[l - 1] &= 0b00111111;
    s.num(F::from_bytes(&or_result).unwrap())
}

impl<F: LurkField> CoCircuit<F> for OrCoprocessor<F> {
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
        synthesize_or(cs, g, s, args)
    }
}

impl<F: LurkField> Coprocessor<F> for OrCoprocessor<F> {
    fn eval_arity(&self) -> usize {
        self.n
    }

    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        compute_or(s, &args)
    }
}

impl<F: LurkField> OrCoprocessor<F> {
    pub fn new() -> Self {
        Self {
            n: 2,
            _p: Default::default(),
        }
    }
}

#[derive(Clone, Debug, Coproc, Serialize, Deserialize)]
pub enum OrCoproc<F: LurkField> {
    SC(OrCoprocessor<F>),
}
