use bellpepper_core::{
    boolean::AllocatedBit, ConstraintSystem, SynthesisError,
};
use core::ops::{AddAssign, MulAssign};
use std::marker::PhantomData;
use bellpepper_core::{boolean::Boolean, num::AllocatedNum};
use ff::{
    derive::byteorder::{ByteOrder, LittleEndian},
    Field, PrimeField, PrimeFieldBits,
};
use super::ecc::AllocatedPoint;
use num_bigint::BigUint;
use pasta_curves::group::Group;
use rand::{RngCore};
use serde::{Deserialize, Serialize};
use sha3::{Digest, Sha3_512};
use lurk_macros::Coproc;
use lurk::{
    circuit::gadgets::{
        pointer::AllocatedPtr,
        constraints::alloc_equal,
    },
    coprocessor::{CoCircuit, Coprocessor},
    field::LurkField,
    lem::pointers::Ptr,
    lem::store::Store,
};
use lazy_static::lazy_static;
use super::utils::{pick, pick_const};

type G1 = halo2curves::grumpkin::G1;

lazy_static! {
    static ref IO_TRUE_HASH: Vec<u8> = hex::decode("4c8ca192c0f6acba0d6816ce095040633a3ef6cb9bcea4f2b834514035f05c1f").unwrap();
    static ref IO_FALSE_HASH: Vec<u8> = hex::decode("032f240cbb095bf8e5a50533e6f86f3695048af3b7279e067364da67c2c8551d").unwrap();
}

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

    let valid = verify_signature(cs, &pk, &r, &s, &c)?;

    let t = AllocatedNum::alloc(cs.namespace(|| "t"), || {
        Ok(F::from_bytes(&IO_TRUE_HASH).unwrap())
    })?;
    let f = AllocatedNum::alloc(cs.namespace(|| "f"), || {
            Ok(F::from_bytes(&IO_FALSE_HASH).unwrap())
    })?;

    let resp = pick(cs.namespace(|| "pick"), &valid, &t, &f)?;

    let t_tag = F::from_u64(2);
    let f_tag = F::from_u64(0);

    let tag = pick_const(cs.namespace(|| "pick_tag"), &valid, t_tag, f_tag)?;

    let tag_type = tag.get_value().unwrap_or(t_tag);

    AllocatedPtr::alloc_tag(
        &mut cs.namespace(|| "output_expr"),
        tag_type,
        resp,
    )
}

fn compute_checksig<F: LurkField>(s: &Store<F>, ptrs: &[Ptr]) -> Ptr {
    let z_ptrs = ptrs.iter().map(|ptr| s.hash_ptr(ptr)).collect::<Vec<_>>();

    let r_x = z_ptrs[0].value().to_bytes();
    let r_y = z_ptrs[1].value().to_bytes();
    let s_bytes = z_ptrs[2].value().to_bytes();
    let pk_x = z_ptrs[3].value().to_bytes();
    let pk_y = z_ptrs[4].value().to_bytes();
    let m_bytes = z_ptrs[5].value().to_bytes();

    let pk_xy = from_xy(pk_x, pk_y);
    let pk = PublicKey::<G1>::from_point(pk_xy);

    let sig_r = from_xy(r_x.clone(), r_y.clone());

    let mut u64_m_array: [u64; 4] = [0; 4];
    for i in 0..4 {
        for j in 0..8 {
            u64_m_array[i] |= (m_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }
    let m = <G1 as Group>::Scalar::from_raw(u64_m_array);

    let mut u64_s_array: [u64; 4] = [0; 4];
    for i in 0..4 {
        for j in 0..8 {
            u64_s_array[i] |= (s_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }
    let sig_s = <G1 as Group>::Scalar::from_raw(u64_s_array);

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
        _g: &lurk::lem::circuit::GlobalAllocator<F>,
        _s: &lurk::lem::store::Store<F>,
        _not_dummy: &Boolean,
        args: &[AllocatedPtr<F>],
    ) -> Result<AllocatedPtr<F>, SynthesisError> {
        synthesize_checksig(cs, args)
    }
}

impl<F: LurkField> Coprocessor<F> for ChecksigCoprocessor<F> {
    fn has_circuit(&self) -> bool {
        true
    }

    fn evaluate_simple(&self, s: &Store<F>, args: &[Ptr]) -> Ptr {
        compute_checksig(s, &args)
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

#[derive(Debug, Clone, Copy)]
pub struct SecretKey<G: Group>(G::Scalar);

impl<G> SecretKey<G>
    where
        G: Group,
{
    #[allow(dead_code)]
    pub fn random(mut rng: impl RngCore) -> Self {
        let secret = G::Scalar::random(&mut rng);
        Self(secret)
    }
}

pub fn from_xy(mut x: Vec<u8>, y: Vec<u8>) -> G1 {
    // Ensure that x has 32 bytes, you might want to handle this differently
    if x.len() != 32 || y.len() != 32 {
        panic!("Input vectors must be exactly 32 bytes long");
    }

    let sign = (y.first().unwrap() & 1) << 6;

    x[31] |= sign;

    let slice = x.as_slice();
    let r: &[u8; 32] = slice.try_into().map_err(|_| "Conversion failed").unwrap();

    return bincode::deserialize(r).unwrap();
}

#[derive(Debug, Clone, Copy)]
pub struct PublicKey<G: Group>(G);

impl<G> PublicKey<G>
    where
        G: Group,
{
    #[allow(dead_code)]
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
    #[allow(dead_code)]
    pub fn sign(self, c: G::Scalar, mut rng: impl RngCore) -> Signature<G> {
        // T
        let mut t = [0u8; 80];
        rng.fill_bytes(&mut t[..]);

        // h = H(T || M)
        let h = Self::hash_to_scalar(b"Nova_Schnorr_Hash", &t[..], c.to_repr().as_mut());

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

    #[allow(dead_code)]
    fn to_uniform(digest: &[u8]) -> G::Scalar {
        assert_eq!(digest.len(), 64);
        let mut bits: [u64; 8] = [0; 8];
        LittleEndian::read_u64_into(digest, &mut bits);
        Self::mul_bits(&G::Scalar::ONE, BitIterator::new(bits))
    }

    #[allow(dead_code)]
    pub fn to_uniform_32(digest: &[u8]) -> G::Scalar {
        assert_eq!(digest.len(), 32);
        let mut bits: [u64; 4] = [0; 4];
        LittleEndian::read_u64_into(digest, &mut bits);
        Self::mul_bits(&G::Scalar::ONE, BitIterator::new(bits))
    }

    #[allow(dead_code)]
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
#[allow(dead_code)]
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
) -> Result<Boolean, SynthesisError> {
    let g = AllocatedPoint::<F>::alloc(
        cs.namespace(|| "g"),
        Some((
            F::from_str_vartime("1").unwrap(),
            F::from_str_vartime(
                "17631683881184975370165255887551781615748388533673675138860",
            ).unwrap(),
            false,
        )),
    )?;

    // g is vesta curve
    let (gx, gy, _) = g.get_coordinates();
    let ex = AllocatedNum::alloc(
        &mut cs.namespace(|| "gx coordinate"),
        || { Ok(F::from_str_vartime("1").unwrap())}
    )?;
    let ey = AllocatedNum::alloc(
        &mut cs.namespace(|| "gy coordinate"),
        || { Ok(F::from_str_vartime("17631683881184975370165255887551781615748388533673675138860").unwrap())}
    )?;

    let gx_equal_bit = alloc_equal(&mut cs.namespace(|| "gx is on curve"), gx, &ex)?;
    let gy_equal_bit = alloc_equal(&mut cs.namespace(|| "gy is on curve"), gy, &ey)?;
    let g_bool: Boolean = Boolean::and(
        &mut cs.namespace(|| "g_and"),
        &gx_equal_bit,
        &gy_equal_bit,
    )?;

    let sg = g.scalar_mul(cs.namespace(|| "[s]G"), s_bits)?;
    let cpk = pk.scalar_mul(&mut cs.namespace(|| "[c]PK"), c_bits)?;
    let rcpk = cpk.add(&mut cs.namespace(|| "R + [c]PK"), r)?;

    let (rcpk_x, rcpk_y, _) = rcpk.get_coordinates();
    let (sg_x, sg_y, _) = sg.get_coordinates();

    let sgx_equal_bit = alloc_equal(&mut cs.namespace(|| "sg_x == rcpk_x"), sg_x, rcpk_x)?;
    let sgy_equal_bit = alloc_equal(&mut cs.namespace(|| "sg_y == rcpk_y"), sg_y, rcpk_y)?;

    let sg_bool: Boolean = Boolean::and(
        &mut cs.namespace(|| "sg_and"),
        &sgx_equal_bit,
        &sgy_equal_bit,
    )?;

    Boolean::and(
        &mut cs.namespace(|| "sig_is_valid"),
        &g_bool,
        &sg_bool,
    )
}

#[cfg(test)]
mod tests {
    use pasta_curves::{
        arithmetic::CurveAffine,
        group::{Group, Curve},
    };
    use halo2curves::bn256::Fr;
    use bellpepper_core::test_cs::TestConstraintSystem;
    use rand_core::OsRng;
    use super::*;
    type G1 = halo2curves::grumpkin::G1;

    #[test]
    fn test_verify() {
        let mut cs = TestConstraintSystem::<Fr>::new();
        assert!(cs.is_satisfied());
        assert_eq!(cs.num_constraints(), 0);

        let sk = SecretKey::<G1>::random(&mut OsRng);
        let hex_string: String = sk.0.to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("sk {}", hex_string);
        //let sk2 = SecretKey::<G1>::random(&mut OsRng);
        //let pkey = PublicKey::from_secret_key(&sk2);
        let pkey = PublicKey::from_secret_key(&sk);

        let pk_bytes = bincode::serialize(&pkey.0).unwrap();

        let hex_string: String = pk_bytes.iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("pk {}", hex_string);

        // generate a random message to sign
        let msg = <G1 as Group>::Scalar::random(&mut OsRng);

        // sign and verify
        let signature = sk.sign(msg, &mut OsRng);
        let sig_r_bytes = bincode::serialize(&signature.r).unwrap();
        let hex_string: String = sig_r_bytes.iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("*sig_r {}", hex_string);
        let hex_string: String = signature.s.to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("*sig_s {}", hex_string);
        let pk_bytes2 = bincode::serialize(&pkey.0).unwrap();
        let hex_string: String = pk_bytes2.iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("*pk {}", hex_string);
        let hex_string: String = msg.to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("*m {}", hex_string);
        let result = pkey.verify(msg, &signature);
        assert!(result);

        // prepare inputs to the circuit gadget
        let pk = {
            let pkxy = pkey.0.to_affine().coordinates().unwrap();

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
        let sig_r_bytes2 = bincode::serialize(&signature.r).unwrap();
        let hex_string: String = sig_r_bytes2.iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("r {}", hex_string);
        let hex_string: String = signature.s.to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
        println!("s {}", hex_string);
        let hex_string: String = msg.to_bytes().iter().rev().map(|byte| format!("{:02x}", byte)).collect();
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
            let c_bits = msg.to_le_bits().iter().map(|b| *b).collect::<Vec<bool>>();

            synthesize_bits2(&mut cs.namespace(|| "c bits"), &Some(c_bits)).unwrap()
        };

        let rxy = signature.r.to_affine().coordinates().unwrap();
        let store = &mut Store::<Fr>::default();
        let rx = store.num(rxy.x().clone());
        let ry = store.num(rxy.y().clone());
        let sbytes = signature.s.to_bytes();
        let ss = store.num(Fr::from_bytes(&sbytes.try_into().unwrap()).unwrap());
        let pkxy = pkey.0.to_affine().coordinates().unwrap();
        let pkx = store.num(pkxy.x().clone());
        let pky = store.num(pkxy.y().clone());
        let mbytes = msg.to_bytes();
        let cc = store.num(Fr::from_bytes(&mbytes.try_into().unwrap()).unwrap());
        let args: &[Ptr] = &[
            rx,
            ry,
            ss,
            pkx,
            pky,
            cc,
        ];
        let valid = compute_checksig(&store, args);
        println!("{:?}", valid);

        // Check the signature was signed by the correct sk using the pk
        let valid2 = verify_signature(&mut cs, &pk, &r, &s, &c).unwrap();
        println!("{:?}", valid2);

        assert!(cs.is_satisfied());
    }
}