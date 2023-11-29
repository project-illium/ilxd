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
    group::{Group, Curve, GroupEncoding},
    arithmetic::CurveAffine,
};
use rand::{rngs::OsRng, RngCore, SeedableRng};
use sha3::{Digest, Sha3_512};
use rand_chacha::ChaChaRng;

type G2 = pasta_curves::vesta::Point;

#[no_mangle]
pub extern "C" fn generate_secret_key(out: *mut u8) {
    // Check if the provided output buffer is null
    if out.is_null() {
        return;
    }

    let sk = SecretKey::<G2>::random(&mut OsRng);
    let sk_bytes = sk.0.to_repr();

    // Copy the secret key bytes to the provided output buffer
    unsafe {
        std::ptr::copy_nonoverlapping(sk_bytes.as_ptr(), out, sk_bytes.len());
    }
}

#[no_mangle]
pub extern "C" fn secret_key_from_seed(seed: *const u8, out: *mut u8) {
    if seed.is_null() || out.is_null() {
        return;
    }

    let mut seed_array: [u8; 32] = [0; 32];
    unsafe {
        std::ptr::copy_nonoverlapping(seed, seed_array.as_mut_ptr(), 32);
    }

    // Create a ChaChaRng from the seed
    let mut rng = ChaChaRng::from_seed(seed_array);

    let sk = SecretKey::<G2>::random(&mut rng);
    let sk_bytes = sk.0.to_repr();

    // Copy the secret key bytes to the provided output buffer
    unsafe {
        std::ptr::copy_nonoverlapping(sk_bytes.as_ptr(), out, sk_bytes.len());
    }
}

#[no_mangle]
pub extern "C" fn priv_to_pub(bytes: *const u8, out: *mut u8) {
    // Ensure that the input bytes slice is valid and has the expected length
    let len = 32;

    // Check if the provided output buffer is null
    if out.is_null() {
        return;
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

    let pk_bytes = pk.0.to_bytes();

    // Copy the public key bytes to the provided output buffer
    unsafe {
        std::ptr::copy_nonoverlapping(pk_bytes.as_ptr(), out, len);
    }
}

#[no_mangle]
pub extern "C" fn compressed_to_full(bytes: *const u8, out_x: *mut u8, out_y: *mut u8) {
    // Ensure that the input bytes slice is valid and has the expected length
    let len = 32;

    // Check if the provided output buffers is null
    if out_x.is_null() {
        return;
    }
    if out_y.is_null() {
        return;
    }

    // Create a byte array from the input bytes
    let mut input_bytes: [u8; 32] = [0; 32];
    unsafe {
        std::ptr::copy_nonoverlapping(bytes, input_bytes.as_mut_ptr(), len);
    }

    let pub_point = G2::from_bytes(&input_bytes).unwrap();
    let pk = PublicKey::from_point(pub_point);

    let xy = pk.0.to_affine().coordinates().unwrap();
    let x_bytes = xy.x().to_repr();
    let y_bytes = xy.y().to_repr();

    // Copy the public key bytes to the provided output buffer
    unsafe {
        std::ptr::copy_nonoverlapping(x_bytes.as_ptr(), out_x, len);
        std::ptr::copy_nonoverlapping(y_bytes.as_ptr(), out_y, len);
    }
}

#[no_mangle]
pub extern "C" fn sign(privkey: *const u8, message_digest: *const u8, out: *mut u8) {
    // Check if the provided output buffer is null
    if out.is_null() {
        return;
    }

    // Assuming privkey and message_digest are always 32 bytes each
    let priv_len = 32;
    let digest_len = 32;

    // Create a byte array from the input private key and message digest
    let mut priv_input_bytes: [u8; 32] = [0; 32];
    let mut m_input_bytes: [u8; 32] = [0; 32];

    unsafe {
        std::ptr::copy_nonoverlapping(privkey, priv_input_bytes.as_mut_ptr(), priv_len);
        std::ptr::copy_nonoverlapping(message_digest, m_input_bytes.as_mut_ptr(), digest_len);
    }

    let mut u64_priv_array: [u64; 4] = [0; 4];
    let mut u64_m_array: [u64; 4] = [0; 4];

    // Use bitwise shifts to convert [u8; 32] to [u64; 4]
    for i in 0..4 {
        for j in 0..8 {
            u64_priv_array[i] |= (priv_input_bytes[i * 8 + j] as u64) << (j * 8);
            u64_m_array[i] |= (m_input_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }

    let b1 = <G2 as Group>::Scalar::from_raw(u64_priv_array);
    let sk = SecretKey::<G2>::from_scalar(b1);
    let m = <G2 as Group>::Scalar::from_raw(u64_m_array);

    let signature = sk.sign(m, &mut OsRng);

    // Serialize the signature into the provided output buffer
    let r = signature.r.to_bytes();
    let s = signature.s.to_repr();

    let mut serialized_data = Vec::new();
    serialized_data.extend_from_slice(&r);
    serialized_data.extend_from_slice(&s);

    // Check if the provided output buffer is large enough for the serialized data
    if serialized_data.len() <= 64 {
        // Copy the serialized data to the provided output buffer
        unsafe {
            std::ptr::copy_nonoverlapping(serialized_data.as_ptr(), out, serialized_data.len());
        }
    }
}

#[no_mangle]
pub extern "C" fn verify(pub_bytes: *const u8, digest_bytes: *const u8, sig_r: *const u8, sig_s: *const u8) -> bool {
    // Ensure that the input bytes slices are valid and have the expected length
    if pub_bytes.is_null() || digest_bytes.is_null() || sig_r.is_null() || sig_s.is_null() {
        return false;
    }

    // Create byte arrays from the input bytes
    let mut pub_input_bytes: [u8; 32] = [0; 32];
    let mut m_bytes: [u8; 32] = [0; 32];
    let mut sig_r_bytes: [u8; 32] = [0; 32];
    let mut sig_s_bytes: [u8; 32] = [0; 32];

    unsafe {
        std::ptr::copy_nonoverlapping(pub_bytes, pub_input_bytes.as_mut_ptr(), 32);
        std::ptr::copy_nonoverlapping(digest_bytes, m_bytes.as_mut_ptr(), 32);
        std::ptr::copy_nonoverlapping(sig_r, sig_r_bytes.as_mut_ptr(), 32);
        std::ptr::copy_nonoverlapping(sig_s, sig_s_bytes.as_mut_ptr(), 32);
    }

    // Convert bytes to the appropriate types
    let pub_point = G2::from_bytes(&pub_input_bytes).unwrap();
    let pk = PublicKey::from_point(pub_point);

    let mut u64_m_array: [u64; 4] = [0; 4];
    for i in 0..4 {
        for j in 0..8 {
            u64_m_array[i] |= (m_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }
    let m = <G2 as Group>::Scalar::from_raw(u64_m_array);

    let r = G2::from_bytes(&sig_r_bytes).unwrap();

    let mut u64_s_array: [u64; 4] = [0; 4];
    for i in 0..4 {
        for j in 0..8 {
            u64_s_array[i] |= (sig_s_bytes[i * 8 + j] as u64) << (j * 8);
        }
    }
    let s = <G2 as Group>::Scalar::from_raw(u64_s_array);

    let signature = Signature { r, s };

    // Verify the signature
    let result = pk.verify(m, &signature);

    result
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
