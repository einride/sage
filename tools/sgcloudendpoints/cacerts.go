package sgcloudendpoints

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgdocker"
)

// espCABundlePath is the path of the CA bundle in the ESPv2 image.
const espCABundlePath = "/etc/ssl/certs/ca-certificates.crt"

// distrustedCAFingerprints returns the SHA-256 fingerprints (upper-case hex of the DER encoding) of the eMudhra
// intermediate CAs with compromised keys, and the roots chaining to them.
// See https://emudhra.we-have-a-problem.com/
func distrustedCAFingerprints() map[string]struct{} {
	return map[string]struct{}{
		// AAA Certificate Services (cross-signer).
		"D7A7A0FB5D7E2731D771E9484EBCDEF71D5F0C3E0A2948782BC83EE0EA699EF4": {},
		// emSign ECC Root CA - G3 (cross-signer).
		"86A1ECBA089C4A8D3BBE2734C612BA341D813E043CF9E8A862CD5C57A36BBE6B": {},
		// emSign Root TLS CA - G3.
		"7DD78D5F4F13459A83DFF9ABBBA62EDBAF6F2D102BF257FD712F4D9F2746ED8D": {},
		// emSign Root CS CA - G3.
		"F2EDAAFF3D9B701FB4D14C9F285F8C0BAD90FC3BCC1FA07D47F4718EDABA34A5": {},
		// EM {DV,OV,EV} TLS CA - G3B, direct and cross-signed.
		"42C4ACCD12C4F59597925E1A08A7A90102608543809308D50EC427E118BA1493": {},
		"45A883F349AEDFD5F3A649B5498FFB121712BA8B448F3F2A91DBC5A766FCDDBE": {},
		"98B896625C0C230ADEEC3EB2CF6AAD5001ECF9156A68644E82374CE5833BC25B": {},
		"5F1DED4DD590AF74631290CF073B122A2AAC6A1FA479778A0BD43579BD832699": {},
		"E7568FE6E12331F4FD7DE61DA4C7559D0F8EB192A218D4741A3BA75184333838": {},
		"579480AEA166290BD3C643589E795FC0D0E54E86505D180F1DFF99EFD3199512": {},
		// EM {DV,OV,EV} TLS CA - G3C, direct and cross-signed.
		"0164D7CEAE5BB909C272AF151FF44B1177AE9B47D9424B94F1D9558C30934D19": {},
		"62B7F9868263F72F6911FC78CEAF72C1979158B7368BF648683BB2AA39B02E72": {},
		"CB6E15165AC802E9B3F993F16C25D41F4E81C21EBDE71AE1B012AC00BF9C7C3F": {},
		"7485B51C0ED192878FA1D2F948FB529AE1842CC5086C4B67F7BE5B2039DE9489": {},
		"432EC25F0D28DE9006D97BF752F17BBB2CD41F669933FCB072D9A1148D0261F3": {},
		"FD9D5C2DBBD4995AB69B8D50C439D9527B00529199C2FD01B2174EB58F07021F": {},
		// EM OV CS CA - G3B and G3C, direct and cross-signed.
		"9E39DE2881273ACC225A149C72BA3E0F51424963669AB633EE1DC7DF5DB067B4": {},
		"A496E23AC0CAB8155FC9B117B97EAE368EFCBB9041AEC8804D5A308D2D01C2D8": {},
		"4B2460A466F95FD40113BCDCCA2CEB29B3B093921DE45474DA7DF48874C630B1": {},
		"84112972B8E59D14560AE7E7119654474AAD119990CF3722BFC1E3B91615E1FD": {},
	}
}

// extractCABundle extracts the CA bundle from the provided image.
// The ESPv2 image is distroless (no shell, no Python), so the bundle is copied out of a stopped container.
func extractCABundle(ctx context.Context, image string) ([]byte, error) {
	var containerID strings.Builder
	create := sgdocker.Command(ctx, "create", image)
	create.Stdout = &containerID
	if err := create.Run(); err != nil {
		return nil, fmt.Errorf("create container from %s: %w", image, err)
	}
	container := strings.TrimSpace(containerID.String())
	defer func() {
		remove := sgdocker.Command(ctx, "rm", container)
		remove.Stdout = nil // suppress noise
		_ = remove.Run()
	}()
	tempDir, err := os.MkdirTemp("", "ca-bundle")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()
	bundleFile := filepath.Join(tempDir, "ca-certificates.crt")
	if err := sgdocker.Command(ctx, "cp", container+":"+espCABundlePath, bundleFile).Run(); err != nil {
		return nil, fmt.Errorf("copy %s from %s: %w", espCABundlePath, image, err)
	}
	bundle, err := os.ReadFile(bundleFile)
	if err != nil {
		return nil, fmt.Errorf("read CA bundle: %w", err)
	}
	return bundle, nil
}

// filterCABundle removes the certificates with a fingerprint in distrusted from the provided PEM bundle.
// It returns the filtered bundle and the number of removed certificates.
func filterCABundle(bundle []byte, distrusted map[string]struct{}) ([]byte, int, error) {
	var filtered bytes.Buffer
	var kept, removed int
	for rest := bundle; len(rest) > 0; {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		fingerprint := sha256.Sum256(block.Bytes)
		if _, ok := distrusted[strings.ToUpper(hex.EncodeToString(fingerprint[:]))]; ok {
			removed++
			continue
		}
		if err := pem.Encode(&filtered, block); err != nil {
			return nil, 0, fmt.Errorf("encode certificate: %w", err)
		}
		kept++
	}
	if kept == 0 {
		return nil, 0, fmt.Errorf("no certificates left in bundle")
	}
	return filtered.Bytes(), removed, nil
}

// writeFilteredCABundle writes the CA bundle of the provided image, without the distrusted CAs, to the provided file.
func writeFilteredCABundle(ctx context.Context, image, outputFile string) error {
	bundle, err := extractCABundle(ctx, image)
	if err != nil {
		return err
	}
	filtered, removed, err := filterCABundle(bundle, distrustedCAFingerprints())
	if err != nil {
		return fmt.Errorf("filter CA bundle of %s: %w", image, err)
	}
	sg.Logger(ctx).Printf("removed %d distrusted CA certificates from %s", removed, image)
	return os.WriteFile(outputFile, filtered, 0o600)
}
