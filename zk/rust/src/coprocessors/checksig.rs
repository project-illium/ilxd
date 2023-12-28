use bellpepper_core::{
    boolean::AllocatedBit, ConstraintSystem, SynthesisError,
};
use core::ops::{AddAssign, MulAssign};
use std::marker::PhantomData;
use bellpepper_core::boolean::Boolean;
use bellpepper_core::num::AllocatedNum;
use ff::{
    derive::byteorder::{ByteOrder, LittleEndian},
    Field, PrimeField, PrimeFieldBits,
};
use crate::coprocessor::ecc::AllocatedPoint;
use num_bigint::BigUint;
use pasta_curves::group::Group;
use rand::{RngCore};
use serde::{Deserialize, Serialize, Serializer};
use sha3::{Digest, Sha3_512};
use lurk_macros::Coproc;
use crate::circuit::gadgets::pointer::AllocatedPtr;
use crate::coprocessor::{CoCircuit, Coprocessor};
use crate::field::LurkField;
use crate::lem::pointers::Ptr;
use crate::lem::store::Store;
use crate::lem::multiframe::MultiFrame;
use crate::tag::{ExprTag, Tag};
use crate::z_ptr::ZPtr;
use crate::{self as lurk};
use pasta_curves::group::GroupEncoding;

type G2 = pasta_curves::vesta::Point;

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ChecksigCoprocessor<F: LurkField> {
    n: usize,
    pub(crate) _p: PhantomData<F>,
}

fn synthesize_checksig<F: LurkField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    ptrs: &[AllocatedPtr<F>],
) -> Result<AllocatedPtr<F>, SynthesisError> {
    let sig_r_x = ptrs[0].hash();
    let sig_r_y = ptrs[1].hash();
    let sig_s_bits = ptrs[2]
        .hash()
        .to_bits_le_strict(&mut cs.namespace(|| "sig_s_bits"))?;
    let pk_x = ptrs[3].hash();
    let pk_y = ptrs[4].hash();
    let c_bits = ptrs[5]
        .hash()
        .to_bits_le_strict(&mut cs.namespace(|| "c_bits"))?;

    let r = AllocatedPoint::<F>::alloc_from_nums(cs.namespace(|| "r"), Some((sig_r_x, sig_r_y, false)))?;
    let pk = AllocatedPoint::<F>::alloc_from_nums(cs.namespace(|| "pk"), Some((pk_x, pk_y, false)))?;
    let s = synthesize_bits(&mut cs.namespace(|| "s bits"), sig_s_bits)?;
    let c = synthesize_bits(&mut cs.namespace(|| "c bits"), c_bits)?;

    let _ = verify_signature(cs, &pk, &r, &s, &c);

    let hex_str = "5698a149855d3a8b3ac99e32b65ce146f00163130070c245a2262b46c5dbc804";
    let t = AllocatedNum::alloc(cs.namespace(|| "t"), || {
        let t_bytes = hex::decode(hex_str).unwrap();
        Ok(F::from_bytes(&t_bytes).unwrap())
    })?;
    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        ExprTag::Sym.to_field(),
        t,
    )
}

fn compute_checksig<F: LurkField, T: Tag>(s: &Store<F>, z_ptrs: &[ZPtr<T, F>]) -> Ptr {
    let r_x = z_ptrs[0].value().to_bytes();
    let r_y = z_ptrs[1].value().to_bytes();
    let s_bytes = z_ptrs[2].value().to_bytes();
    let pk_x = z_ptrs[3].value().to_bytes();
    let pk_y = z_ptrs[4].value().to_bytes();
    let m_bytes = z_ptrs[5].value().to_bytes();

    let pk_xy = from_xy(pk_x, pk_y);
    let pk = PublicKey::<G2>::from_point(pk_xy);

    let sig_r = from_xy(r_x, r_y);

    let mut u64_m_array: [u64; 4] = [0; 4];
    for i in 0..4 {
        for j in 0..8 {
            u64_m_array[i] |= (m_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }
    let m = <G2 as Group>::Scalar::from_raw(u64_m_array);

    let mut u64_s_array: [u64; 4] = [0; 4];
    for i in 0..4 {
        for j in 0..8 {
            u64_s_array[i] |= (s_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }
    let sig_s = <G2 as Group>::Scalar::from_raw(u64_s_array);

    let sig = Signature{r: sig_r, s: sig_s};

    let result = pk.verify(m, &sig);
    if result {
        return s.intern_lurk_symbol("t");
    }
    s.intern_nil()
}

impl<F: LurkField> CoCircuit<F> for ChecksigCoprocessor<F> {
    fn arity(&self) -> usize {
        self.n
    }

    #[inline]
    fn synthesize_simple<CS: ConstraintSystem<F>>(
        &self,
        cs: &mut CS,
        _g: &crate::lem::circuit::GlobalAllocator<F>,
        _s: &crate::lem::store::Store<F>,
        _not_dummy: &Boolean,
        args: &[AllocatedPtr<F>],
    ) -> Result<AllocatedPtr<F>, SynthesisError> {
        synthesize_checksig(cs, args)
    }
}

impl<F: LurkField> Coprocessor<F> for ChecksigCoprocessor<F> {
    fn eval_arity(&self) -> usize {
        self.n
    }

    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        let z_ptrs = args.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();
        compute_checksig(s, &z_ptrs)
    }
}

impl<F: LurkField> ChecksigCoprocessor<F> {
    pub fn new() -> Self {
        Self {
            n: 6,
            _p: Default::default(),
        }
    }
}

#[derive(Clone, Debug, Coproc, Serialize, Deserialize)]
pub enum ChecksigCoproc<F: LurkField> {
    SC(ChecksigCoprocessor<F>),
}

impl<'a, Fq> Serialize for MultiFrame<'a, Fq, ChecksigCoproc<Fq>> where Fq: Serialize + LurkField {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
    {
        // This is a dummy implementation that does nothing
        serializer.serialize_unit()
    }
}

#[derive(Debug, Clone, Copy)]
pub struct SecretKey<G: Group>(G::Scalar);

impl<G> SecretKey<G>
    where
        G: Group,
{
    pub fn random(mut rng: impl RngCore) -> Self {
        let secret = G::Scalar::random(&mut rng);
        Self(secret)
    }
}

pub fn from_xy(mut x: Vec<u8>, y: Vec<u8>) -> G2 {
    // Ensure that x has 32 bytes, you might want to handle this differently
    if x.len() != 32 || y.len() != 32 {
        panic!("Input vectors must be exactly 32 bytes long");
    }

    let sign = (y.first().unwrap() & 1) << 7;

    x[31] |= sign;

    let slice = x.as_slice();
    let r: &[u8; 32] = slice.try_into().map_err(|_| "Conversion failed").unwrap();

    G2::from_bytes(r).unwrap()
}

#[derive(Debug, Clone, Copy)]
pub struct PublicKey<G: Group>(G);

impl<G> PublicKey<G>
    where
        G: Group,
{
    pub fn from_secret_key(s: &SecretKey<G>) -> Self {
        let point = G::generator() * s.0;
        Self(point)
    }

    pub fn from_point(point: G) -> Self{
        Self(point)
    }
}

#[derive(Clone)]
pub struct Signature<G: Group> {
    pub r: G,
    pub s: G::Scalar,
}

impl<G> SecretKey<G>
    where
        G: Group,
{
    pub fn sign(self, c: G::Scalar, mut rng: impl RngCore) -> Signature<G> {
        // T
        let mut t = [0u8; 80];
        rng.fill_bytes(&mut t[..]);

        // h = H(T || M)
        let h = Self::hash_to_scalar(b"Nova_Ecdsa_Hash", &t[..], c.to_repr().as_mut());

        // R = [h]G
        let r = G::generator().mul(h);

        // s = h + c * sk
        let mut s = c;

        s.mul_assign(&self.0);
        s.add_assign(&h);

        Signature { r, s }
    }

    fn mul_bits<B: AsRef<[u64]>>(s: &G::Scalar, bits: BitIterator<B>) -> G::Scalar {
        let mut x = G::Scalar::ZERO;
        for bit in bits {
            x = x.double();

            if bit {
                x.add_assign(s)
            }
        }
        x
    }

    fn to_uniform(digest: &[u8]) -> G::Scalar {
        assert_eq!(digest.len(), 64);
        let mut bits: [u64; 8] = [0; 8];
        LittleEndian::read_u64_into(digest, &mut bits);
        Self::mul_bits(&G::Scalar::ONE, BitIterator::new(bits))
    }

    pub fn to_uniform_32(digest: &[u8]) -> G::Scalar {
        assert_eq!(digest.len(), 32);
        let mut bits: [u64; 4] = [0; 4];
        LittleEndian::read_u64_into(digest, &mut bits);
        Self::mul_bits(&G::Scalar::ONE, BitIterator::new(bits))
    }

    pub fn hash_to_scalar(persona: &[u8], a: &[u8], b: &[u8]) -> G::Scalar {
        let mut hasher = Sha3_512::new();
        hasher.update(persona);
        hasher.update(a);
        hasher.update(b);
        let digest = hasher.finalize();
        Self::to_uniform(digest.as_ref())
    }
}

impl<G> PublicKey<G>
    where
        G: Group,
        G::Scalar: PrimeFieldBits,
{
    pub fn verify(&self, c: G::Scalar, signature: &Signature<G>) -> bool {
        let modulus = Self::modulus_as_scalar();
        let order_check_pk = self.0.mul(modulus);
        if !order_check_pk.eq(&G::identity()) {
            return false;
        }

        let order_check_r = signature.r.mul(modulus);
        if !order_check_r.eq(&G::identity()) {
            return false;
        }

        // 0 = [-s]G + R + [c]PK
        self
            .0
            .mul(c)
            .add(&signature.r)
            .add(G::generator().mul(signature.s).neg())
            .eq(&G::identity())
    }

    fn modulus_as_scalar() -> G::Scalar {
        let mut bits = G::Scalar::char_le_bits().to_bitvec();
        let mut acc = BigUint::new(Vec::<u32>::new());
        while let Some(b) = bits.pop() {
            acc <<= 1_i32;
            acc += u8::from(b);
        }
        let modulus = acc.to_str_radix(10);
        G::Scalar::from_str_vartime(&modulus).unwrap()
    }
}

#[derive(Debug)]
pub struct BitIterator<F> {
    t: F,
    n: usize,
}

impl<F: AsRef<[u64]>> BitIterator<F> {
    pub fn new(t: F) -> Self {
        let n = t.as_ref().len() * 64;

        BitIterator { t, n }
    }
}

impl<F: AsRef<[u64]>> Iterator for BitIterator<F> {
    type Item = bool;

    fn next(&mut self) -> Option<bool> {
        if self.n == 0 {
            None
        } else {
            self.n -= 1;
            let part = self.n / 64;
            let bit = self.n - (64 * part);

            Some(self.t.as_ref()[part] & (1 << bit) > 0)
        }
    }
}

// Synthesize a bit representation into circuit gadgets.
pub fn synthesize_bits<F: PrimeField, CS: ConstraintSystem<F>>(
    _cs: &mut CS,
    bits: Vec<Boolean>,
) -> Result<Vec<AllocatedBit>, SynthesisError> {
    bits.iter().map(|bit| {
        match bit {
            Boolean::Is(allocated_bit) => Ok(allocated_bit.clone()),
            Boolean::Not(_allocated_bit) => {
                Err(SynthesisError::AssignmentMissing)
            },
            Boolean::Constant(_value) => {
                Err(SynthesisError::AssignmentMissing)
            }
        }
    }).collect::<Result<Vec<AllocatedBit>, SynthesisError>>()
}

// Synthesize a bit representation into circuit gadgets.
pub fn synthesize_bits2<F: PrimeField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    bits: &Option<Vec<bool>>,
) -> Result<Vec<AllocatedBit>, SynthesisError> {
    (0..F::NUM_BITS)
        .map(|i| {
            AllocatedBit::alloc(
                cs.namespace(|| format!("bit {i}")),
                Some(bits.as_ref().unwrap()[i as usize]),
            )
        })
        .collect::<Result<Vec<AllocatedBit>, SynthesisError>>()
}

pub fn verify_signature<F: PrimeField, CS: ConstraintSystem<F>>(
    cs: &mut CS,
    pk: &AllocatedPoint<F>,
    r: &AllocatedPoint<F>,
    s_bits: &[AllocatedBit],
    c_bits: &[AllocatedBit],
) -> Result<(), SynthesisError> {
    let g = AllocatedPoint::<F>::alloc(
        cs.namespace(|| "g"),
        Some((
            F::from_str_vartime(
                "28948022309329048855892746252171976963363056481941647379679742748393362948096",
            )
                .unwrap(),
            F::from_str_vartime("2").unwrap(),
            false,
        )),
    )?;

    cs.enforce(
        || "gx is vesta curve",
        |lc| lc + g.get_coordinates().0.get_variable(),
        |lc| lc + CS::one(),
        |lc| {
            lc + (
                F::from_str_vartime(
                    "28948022309329048855892746252171976963363056481941647379679742748393362948096",
                )
                    .unwrap(),
                CS::one(),
            )
        },
    );

    cs.enforce(
        || "gy is vesta curve",
        |lc| lc + g.get_coordinates().1.get_variable(),
        |lc| lc + CS::one(),
        |lc| lc + (F::from_str_vartime("2").unwrap(), CS::one()),
    );

    let sg = g.scalar_mul(cs.namespace(|| "[s]G"), s_bits)?;
    let cpk = pk.scalar_mul(&mut cs.namespace(|| "[c]PK"), c_bits)?;
    let rcpk = cpk.add(&mut cs.namespace(|| "R + [c]PK"), r)?;

    let (rcpk_x, rcpk_y, _) = rcpk.get_coordinates();
    let (sg_x, sg_y, _) = sg.get_coordinates();

    cs.enforce(
        || "sg_x == rcpk_x",
        |lc| lc + sg_x.get_variable(),
        |lc| lc + CS::one(),
        |lc| lc + rcpk_x.get_variable(),
    );

    cs.enforce(
        || "sg_y == rcpk_y",
        |lc| lc + sg_y.get_variable(),
        |lc| lc + CS::one(),
        |lc| lc + rcpk_y.get_variable(),
    );

    Ok(())
}

#[cfg(test)]
mod tests {
    use pasta_curves::{
        arithmetic::CurveAffine,
        pallas::Scalar as Fr,
        group::{Group, Curve},
    };
    use bellpepper_core::test_cs::TestConstraintSystem;
    use rand_core::OsRng;
    use super::*;
    type G2 = pasta_curves::vesta::Point;

    #[test]
    fn test_verify() {
        let mut cs = TestConstraintSystem::<Fr>::new();
        assert!(cs.is_satisfied());
        assert_eq!(cs.num_constraints(), 0);

        let sk = SecretKey::<G2>::random(&mut OsRng);
        //let sk2 = SecretKey::<G2>::random(&mut OsRng);
        //let pk = PublicKey::from_secret_key(&sk2);
        let pk = PublicKey::from_secret_key(&sk);

        // generate a random message to sign
        let c = <G2 as Group>::Scalar::random(&mut OsRng);

        // sign and verify
        let signature = sk.sign(c, &mut OsRng);
        let result = pk.verify(c, &signature);
        //assert!(result);

        // prepare inputs to the circuit gadget
        let pk = {
            let pkxy = pk.0.to_affine().coordinates().unwrap();

            let hex_string: String = pkxy.x().to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
            println!("pkx {}", hex_string);
            let hex_string: String = pkxy.y().to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
            println!("pky {}", hex_string);

            AllocatedPoint::<Fr>::alloc(
                cs.namespace(|| "pub key"),
                Some((*pkxy.x(), *pkxy.y(), false)),
            )
                .unwrap()
        };
        let r = {
            let rxy = signature.r.to_affine().coordinates().unwrap();
            let hex_string: String = rxy.x().to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
            println!("rx {}", hex_string);
            let hex_string: String = rxy.y().to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
            println!("ry {}", hex_string);
            AllocatedPoint::alloc(cs.namespace(|| "r"), Some((*rxy.x(), *rxy.y(), false))).unwrap()
        };
        let hex_string: String = signature.s.to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("s {}", hex_string);
        let hex_string: String = c.to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("c {}", hex_string);
        let s = {
            let s_bits = signature
                .s
                .to_le_bits()
                .iter()
                .map(|b| *b)
                .collect::<Vec<bool>>();

            synthesize_bits2(&mut cs.namespace(|| "s bits"), &Some(s_bits)).unwrap()
        };
        let c = {
            let c_bits = c.to_le_bits().iter().map(|b| *b).collect::<Vec<bool>>();

            synthesize_bits2(&mut cs.namespace(|| "c bits"), &Some(c_bits)).unwrap()
        };

        // Check the signature was signed by the correct sk using the pk
        verify_signature(&mut cs, &pk, &r, &s, &c).unwrap();

        assert!(cs.is_satisfied());
    }
}