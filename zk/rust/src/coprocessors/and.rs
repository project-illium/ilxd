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
pub struct AndCoprocessor<F: LurkField> {
    n: usize,
    pub(crate) _p: PhantomData<F>,
}

fn synthesize_and<F: LurkField, CS: ConstraintSystem<F>>(
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

    let mut and_result = Vec::new();
    for i in 0..hash_a.len() {
        let and_element: Boolean = Boolean::and(
            &mut cs.namespace(|| format!("and_{}", i)),
            hash_a.get(i).unwrap(),
            hash_b.get(i).unwrap(),
        )?;
        and_result.push(and_element);
    }

    let x_scalar = pack_bits(cs.namespace(|| "a_scalar"), &and_result)?;
    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        F::from(4),
        x_scalar,
    )
}

fn compute_and<F: LurkField>(s: &Store<F>, ptrs: &[Ptr]) -> Ptr {
    let z_ptrs = ptrs.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();

    let hash_a = z_ptrs[0].value().to_bytes();
    let hash_b = z_ptrs[1].value().to_bytes();

    let mut and_result: Vec<u8> = hash_a.iter()
        .zip(hash_b.iter())
        .map(|(&x, &y)| x & y)
        .collect();

    let l = and_result.len();

    and_result[l - 1] &= 0b00111111;
    s.num(F::from_bytes(&and_result).unwrap())
}

impl<F: LurkField> CoCircuit<F> for AndCoprocessor<F> {
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
        synthesize_and(cs, g, s, args)
    }
}

impl<F: LurkField> Coprocessor<F> for AndCoprocessor<F> {
    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        compute_and(s, &args)
    }
}

impl<F: LurkField> AndCoprocessor<F> {
    pub fn new() -> Self {
        Self {
            n: 2,
            _p: Default::default(),
        }
    }
}

#[derive(Clone, Debug, Coproc, Serialize, Deserialize)]
pub enum AndCoproc<F: LurkField> {
    SC(AndCoprocessor<F>),
}
