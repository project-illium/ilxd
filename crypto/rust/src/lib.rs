// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

use core::ops::{AddAssign, MulAssign};
use ff::{
    derive::byteorder::{ByteOrder, LittleEndian},
    Field, PrimeField, PrimeFieldBits,
};
use num_bigint::BigUint;
use pasta_curves::{
    group::{Group, Curve},
    arithmetic::CurveAffine,
};
use rand::{rngs::OsRng, RngCore};
use sha3::{Digest, Sha3_512};
//use std::os::raw::c_void;

#[repr(C)]
pub struct KeyBytes {
    data: *const u8,
    len: usize,
}

#[no_mangle]
pub extern "C" fn generate_secret_key() -> KeyBytes {
    let sk = SecretKey::<G2>::random(&mut OsRng);
    let sk_bytes = sk.0.to_repr();

    let sk_len = sk_bytes.len();
    let sk_ptr = Box::into_raw(Box::new(sk_bytes)) as *const u8;

    KeyBytes {
        data: sk_ptr,
        len: sk_len,
    }
}

#[no_mangle]
pub extern "C" fn priv_to_pub(bytes: *const u8, len: usize) -> KeyBytes {
    // Ensure that the input bytes slice is valid and has the expected length
    if len != 32 {
        // Return an empty KeyBytes with a null pointer if the length is incorrect
        return KeyBytes {
            data: std::ptr::null(),
            len: 0,
        };
    }

    // Create a byte array from the input bytes
    let mut input_bytes: [u8; 32] = [0; 32];
    unsafe {
        std::ptr::copy_nonoverlapping(bytes, input_bytes.as_mut_ptr(), len);
    }

    let mut u64_array: [u64; 4] = [0; 4];

    // Use bitwise shifts to convert [u8; 32] to [u64; 4]
    for i in 0..4 {
        for j in 0..8 {
            u64_array[i] |= (input_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }

    let b = <G2 as Group>::Scalar::from_raw(u64_array);
    let sk = SecretKey::<G2>::from_scalar(b);
    let pk = PublicKey::from_secret_key(&sk);

    let pkxy = pk.0.to_affine().coordinates().unwrap();
    let x = pkxy.x().to_repr();
    let y = pkxy.y().to_repr();

    // Serialize x and y into a Vec<u8>
    let mut serialized_data = Vec::new();
    serialized_data.extend_from_slice(&x);
    serialized_data.extend_from_slice(&y);

    // Allocate memory for the serialized data
    let serialized_len = serialized_data.len();
    let serialized_ptr = serialized_data.as_ptr();

    // Ensure the serialized data lives as long as the KeyBytes object
    std::mem::forget(serialized_data);

    KeyBytes {
        data: serialized_ptr,
        len: serialized_len,
    }
}

#[no_mangle]
pub extern "C" fn free_memory(ptr: *mut u8) {
    // Safety: We assume that `ptr` is a valid pointer to memory
    // allocated with `malloc` or a compatible allocator.
    unsafe {
        // Deallocate the memory pointed to by `ptr`
        libc::free(ptr as *mut libc::c_void);
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

    pub fn from_scalar(secret : G::Scalar) -> Self {
        Self(secret)
    }
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
pub struct BitIterator<E> {
    t: E,
    n: usize,
}

impl<E: AsRef<[u64]>> BitIterator<E> {
    pub fn new(t: E) -> Self {
        let n = t.as_ref().len() * 64;

        BitIterator { t, n }
    }
}

impl<E: AsRef<[u64]>> Iterator for BitIterator<E> {
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

//type G1 = pasta_curves::pallas::Point;
type G2 = pasta_curves::vesta::Point;

/*

[[bin]]
name = "illium_crypto"  # Specify the name of your binary
path = "src/lib.rs"  # Path to your main Rust source file

fn main() {

    let sk = SecretKey::<G2>::random(&mut OsRng);
    let pk = PublicKey::from_secret_key(&sk);

    // generate a random message to sign
    let c = <G2 as Group>::Scalar::random(&mut OsRng);

    // sign and verify
    let signature = sk.sign(c, &mut OsRng);
    let result = pk.verify(c, &signature);
    assert!(result);

    println!("here");
}*/