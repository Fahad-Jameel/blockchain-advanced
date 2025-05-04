package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
)

// VRFOracle implements verifiable random functions
type VRFOracle struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

// NewVRFOracle creates a new VRF oracle
func NewVRFOracle() *VRFOracle {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	return &VRFOracle{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}
}

// Generate generates a VRF output and proof
func (vrf *VRFOracle) Generate(seed uint64) ([]byte, []byte) {
	// Convert seed to bytes
	seedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seedBytes, seed)

	// Hash the seed to get a point on the curve
	hash := sha256.Sum256(seedBytes)
	hx, hy := hashToCurve(hash[:])

	// Calculate gamma = sk * H(seed)
	gammaX, gammaY := vrf.privateKey.Curve.ScalarMult(hx, hy, vrf.privateKey.D.Bytes())

	// Generate proof
	proof := vrf.generateProof(seedBytes, gammaX, gammaY)

	// Calculate VRF output
	output := sha256.Sum256(append(gammaX.Bytes(), gammaY.Bytes()...))

	// Serialize proof
	proofBytes := vrf.serializeProof(proof)

	return output[:], proofBytes
}

// generateProof generates a VRF proof
func (vrf *VRFOracle) generateProof(seed []byte, gammaX, gammaY *big.Int) *VRFProof {
	// Generate random k
	k, _ := rand.Int(rand.Reader, vrf.privateKey.Curve.Params().N)

	// Calculate u = g^k
	ux, uy := vrf.privateKey.Curve.ScalarBaseMult(k.Bytes())

	// Calculate v = h^k
	hash := sha256.Sum256(seed)
	hx, hy := hashToCurve(hash[:])
	vx, vy := vrf.privateKey.Curve.ScalarMult(hx, hy, k.Bytes())

	// Calculate challenge c = H(g, h, pk, gamma, u, v)
	c := vrf.calculateChallenge(hx, hy, gammaX, gammaY, ux, uy, vx, vy)

	// Calculate response s = k - c * sk mod n
	s := new(big.Int).Mul(c, vrf.privateKey.D)
	s.Sub(k, s)
	s.Mod(s, vrf.privateKey.Curve.Params().N)

	return &VRFProof{
		Gamma:     &ECPoint{X: gammaX, Y: gammaY},
		C:         c,
		S:         s,
		PublicKey: vrf.publicKey,
	}
}

// Helper types and functions

type VRFProof struct {
	Gamma     *ECPoint
	C         *big.Int
	S         *big.Int
	PublicKey *ecdsa.PublicKey
}

type ECPoint struct {
	X *big.Int
	Y *big.Int
}

func hashToCurve(data []byte) (*big.Int, *big.Int) {
	// Simple hash-to-curve implementation
	hash := sha256.Sum256(data)
	x := new(big.Int).SetBytes(hash[:])

	curve := elliptic.P256()
	params := curve.Params()

	// Find a valid point on the curve
	for {
		// Calculate y^2 = x^3 - 3x + b
		y2 := new(big.Int).Mul(x, x)
		y2.Mul(y2, x)

		threeX := new(big.Int).Mul(x, big.NewInt(3))
		y2.Sub(y2, threeX)
		y2.Add(y2, params.B)
		y2.Mod(y2, params.P)

		// Try to find square root
		y := new(big.Int).ModSqrt(y2, params.P)
		if y != nil {
			return x, y
		}

		x.Add(x, big.NewInt(1))
		x.Mod(x, params.P)
	}
}

func (vrf *VRFOracle) calculateChallenge(hx, hy, gammaX, gammaY, ux, uy, vx, vy *big.Int) *big.Int {
	// H(g, h, pk, gamma, u, v)
	data := append(vrf.publicKey.X.Bytes(), vrf.publicKey.Y.Bytes()...)
	data = append(data, hx.Bytes()...)
	data = append(data, hy.Bytes()...)
	data = append(data, gammaX.Bytes()...)
	data = append(data, gammaY.Bytes()...)
	data = append(data, ux.Bytes()...)
	data = append(data, uy.Bytes()...)
	data = append(data, vx.Bytes()...)
	data = append(data, vy.Bytes()...)

	hash := sha256.Sum256(data)
	return new(big.Int).SetBytes(hash[:])
}

func (vrf *VRFOracle) serializeProof(proof *VRFProof) []byte {
	var data []byte
	data = append(data, proof.Gamma.X.Bytes()...)
	data = append(data, proof.Gamma.Y.Bytes()...)
	data = append(data, proof.C.Bytes()...)
	data = append(data, proof.S.Bytes()...)
	return data
}
