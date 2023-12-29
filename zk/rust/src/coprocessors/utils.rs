//! This module implements various low-level gadgets
use bellpepper::gadgets::Assignment;
use bellpepper_core::{
    boolean::{AllocatedBit, Boolean},
    num::AllocatedNum,
    ConstraintSystem, LinearCombination, SynthesisError,
};
use ff::{PrimeField, PrimeFieldBits};
use itertools::Itertools as _;

/// Gets as input the little indian representation of a number and spits out the number
pub fn le_bits_to_num<Scalar, CS>(
    mut cs: CS,
    bits: &[AllocatedBit],
) -> Result<AllocatedNum<Scalar>, SynthesisError>
    where
        Scalar: PrimeField + PrimeFieldBits,
        CS: ConstraintSystem<Scalar>,
{
    // We loop over the input bits and construct the constraint
    // and the field element that corresponds to the result
    let mut lc = LinearCombination::zero();
    let mut coeff = Scalar::ONE;
    let mut fe = Some(Scalar::ZERO);
    for bit in bits.iter() {
        lc = lc + (coeff, bit.get_variable());
        fe = bit.get_value().map(|val| {
            if val {
                fe.unwrap() + coeff
            } else {
                fe.unwrap()
            }
        });
        coeff = coeff.double();
    }
    let num = AllocatedNum::alloc(cs.namespace(|| "Field element"), || {
        fe.ok_or(SynthesisError::AssignmentMissing)
    })?;
    lc = lc - num.get_variable();
    cs.enforce(|| "compute number from bits", |lc| lc, |lc| lc, |_| lc);
    Ok(num)
}

/// Allocate a variable that is set to zero
pub fn alloc_zero<F: PrimeField, CS: ConstraintSystem<F>>(mut cs: CS) -> AllocatedNum<F> {
    let zero = AllocatedNum::alloc_infallible(cs.namespace(|| "alloc"), || F::ZERO);
    cs.enforce(
        || "check zero is valid",
        |lc| lc,
        |lc| lc,
        |lc| lc + zero.get_variable(),
    );
    zero
}

/// Allocate a variable that is set to one
pub fn alloc_one<F: PrimeField, CS: ConstraintSystem<F>>(mut cs: CS) -> AllocatedNum<F> {
    let one = AllocatedNum::alloc_infallible(cs.namespace(|| "alloc"), || F::ONE);
    cs.enforce(
        || "check one is valid",
        |lc| lc + CS::one(),
        |lc| lc + CS::one(),
        |lc| lc + one.get_variable(),
    );

    one
}

/// Check that two numbers are equal and return a bit
pub fn alloc_num_equals<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    b: &AllocatedNum<F>,
) -> Result<AllocatedBit, SynthesisError> {
    // Allocate and constrain `r`: result boolean bit.
    // It equals `true` if `a` equals `b`, `false` otherwise
    let r_value = match (a.get_value(), b.get_value()) {
        (Some(a), Some(b)) => Some(a == b),
        _ => None,
    };

    let r = AllocatedBit::alloc(cs.namespace(|| "r"), r_value)?;

    // Allocate t s.t. t=1 if a == b else 1/(a - b)

    let t = AllocatedNum::alloc(cs.namespace(|| "t"), || {
        let a_val = *a.get_value().get()?;
        let b_val = *b.get_value().get()?;
        Ok(if a_val == b_val {
            F::ONE
        } else {
            (a_val - b_val).invert().unwrap()
        })
    })?;

    cs.enforce(
        || "t*(a - b) = 1 - r",
        |lc| lc + t.get_variable(),
        |lc| lc + a.get_variable() - b.get_variable(),
        |lc| lc + CS::one() - r.get_variable(),
    );

    cs.enforce(
        || "r*(a - b) = 0",
        |lc| lc + r.get_variable(),
        |lc| lc + a.get_variable() - b.get_variable(),
        |lc| lc,
    );

    Ok(r)
}

/// If condition return a otherwise b
pub fn conditionally_select<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    b: &AllocatedNum<F>,
    condition: &Boolean,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? {
            Ok(*a.get_value().get()?)
        } else {
            Ok(*b.get_value().get()?)
        }
    })?;

    // a * condition + b*(1-condition) = c ->
    // a * condition - b*condition = c - b
    cs.enforce(
        || "conditional select constraint",
        |lc| lc + a.get_variable() - b.get_variable(),
        |_| condition.lc(CS::one(), F::ONE),
        |lc| lc + c.get_variable() - b.get_variable(),
    );

    Ok(c)
}

/// If condition return a otherwise b
pub fn conditionally_select_vec<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &[AllocatedNum<F>],
    b: &[AllocatedNum<F>],
    condition: &Boolean,
) -> Result<Vec<AllocatedNum<F>>, SynthesisError> {
    a.iter()
        .zip_eq(b.iter())
        .enumerate()
        .map(|(i, (a, b))| {
            conditionally_select(cs.namespace(|| format!("select_{i}")), a, b, condition)
        })
        .collect::<Result<Vec<AllocatedNum<F>>, SynthesisError>>()
}

/// Same as the above but Condition is an `AllocatedNum` that needs to be
/// 0 or 1. 1 => True, 0 => False
pub fn conditionally_select2<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    b: &AllocatedNum<F>,
    condition: &AllocatedNum<F>,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? == F::ONE {
            Ok(*a.get_value().get()?)
        } else {
            Ok(*b.get_value().get()?)
        }
    })?;

    // a * condition + b*(1-condition) = c ->
    // a * condition - b*condition = c - b
    cs.enforce(
        || "conditional select constraint",
        |lc| lc + a.get_variable() - b.get_variable(),
        |lc| lc + condition.get_variable(),
        |lc| lc + c.get_variable() - b.get_variable(),
    );

    Ok(c)
}

/// If condition set to 0 otherwise a. Condition is an allocated num
pub fn select_zero_or_num2<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    condition: &AllocatedNum<F>,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? == F::ONE {
            Ok(F::ZERO)
        } else {
            Ok(*a.get_value().get()?)
        }
    })?;

    // a * (1 - condition) = c
    cs.enforce(
        || "conditional select constraint",
        |lc| lc + a.get_variable(),
        |lc| lc + CS::one() - condition.get_variable(),
        |lc| lc + c.get_variable(),
    );

    Ok(c)
}

/// If condition set to a otherwise 0. Condition is an allocated num
pub fn select_num_or_zero2<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    condition: &AllocatedNum<F>,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? == F::ONE {
            Ok(*a.get_value().get()?)
        } else {
            Ok(F::ZERO)
        }
    })?;

    cs.enforce(
        || "conditional select constraint",
        |lc| lc + a.get_variable(),
        |lc| lc + condition.get_variable(),
        |lc| lc + c.get_variable(),
    );

    Ok(c)
}

/// If condition set to a otherwise 0
pub fn select_num_or_zero<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    condition: &Boolean,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? {
            Ok(*a.get_value().get()?)
        } else {
            Ok(F::ZERO)
        }
    })?;

    cs.enforce(
        || "conditional select constraint",
        |lc| lc + a.get_variable(),
        |_| condition.lc(CS::one(), F::ONE),
        |lc| lc + c.get_variable(),
    );

    Ok(c)
}

/// If condition set to 1 otherwise a
pub fn select_one_or_num2<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    condition: &AllocatedNum<F>,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? == F::ONE {
            Ok(F::ONE)
        } else {
            Ok(*a.get_value().get()?)
        }
    })?;

    cs.enforce(
        || "conditional select constraint",
        |lc| lc + CS::one() - a.get_variable(),
        |lc| lc + condition.get_variable(),
        |lc| lc + c.get_variable() - a.get_variable(),
    );
    Ok(c)
}

/// If condition set to 1 otherwise a - b
pub fn select_one_or_diff2<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    b: &AllocatedNum<F>,
    condition: &AllocatedNum<F>,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? == F::ONE {
            Ok(F::ONE)
        } else {
            Ok(*a.get_value().get()? - *b.get_value().get()?)
        }
    })?;

    cs.enforce(
        || "conditional select constraint",
        |lc| lc + CS::one() - a.get_variable() + b.get_variable(),
        |lc| lc + condition.get_variable(),
        |lc| lc + c.get_variable() - a.get_variable() + b.get_variable(),
    );
    Ok(c)
}

/// If condition set to a otherwise 1 for boolean conditions
pub fn select_num_or_one<F: PrimeField, CS: ConstraintSystem<F>>(
    mut cs: CS,
    a: &AllocatedNum<F>,
    condition: &Boolean,
) -> Result<AllocatedNum<F>, SynthesisError> {
    let c = AllocatedNum::alloc(cs.namespace(|| "conditional select result"), || {
        if *condition.get_value().get()? {
            Ok(*a.get_value().get()?)
        } else {
            Ok(F::ONE)
        }
    })?;

    cs.enforce(
        || "conditional select constraint",
        |lc| lc + a.get_variable() - CS::one(),
        |_| condition.lc(CS::one(), F::ONE),
        |lc| lc + c.get_variable() - CS::one(),
    );

    Ok(c)
}