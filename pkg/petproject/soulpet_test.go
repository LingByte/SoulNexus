package petproject

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestValidateSoulpetPackage_sprite(t *testing.T) {
	files := map[string]string{
		SoulpetYamlFile: "specVersion: 1\nname: Test\nkind: sprite\n",
		ManifestFile: `{
			"version": 1,
			"type": "sprite",
			"name": "Test",
			"assets": {
				"sprite": {
					"baseUrl": "assets/sprites/",
					"animations": {
						"idle": { "files": ["a.png"], "frameWidth": 64, "frameHeight": 64, "frames": 1 }
					}
				}
			}
		}`,
		DefaultEntry: "// pet ok",
	}
	kind, issues := ValidateSoulpetPackage(files, nil)
	if kind != KindSprite {
		t.Fatalf("kind=%q", kind)
	}
	if HasValidationErrors(issues) {
		t.Fatalf("unexpected errors: %+v", issues)
	}
}

func TestValidateSoulpetPackage_live2d(t *testing.T) {
	files := map[string]string{
		SoulpetYamlFile: "specVersion: 1\nname: L2D\nkind: live2d\n",
		ManifestFile: `{
			"version": 1,
			"type": "live2d",
			"name": "L2D",
			"assets": {
				"live2d": {
					"baseUrl": "assets/live2d/",
					"model": "model.model3.json",
					"motions": { "idle": "motions/idle.motion3.json" }
				}
			}
		}`,
		DefaultEntry: "// live2d ok",
	}
	kind, issues := ValidateSoulpetPackage(files, nil)
	if kind != KindLive2D {
		t.Fatalf("kind=%q", kind)
	}
	if HasValidationErrors(issues) {
		t.Fatalf("unexpected errors: %+v", issues)
	}
}

func TestPackUnpackZip(t *testing.T) {
	files := map[string]string{
		SoulpetYamlFile: "specVersion: 1\nname: Zip\nkind: sprite\n",
		ManifestFile:    `{"version":1,"type":"sprite","name":"Zip","assets":{"sprite":{"baseUrl":"assets/","animations":{"idle":{"files":["a.png"],"frameWidth":1,"frameHeight":1,"frames":1}}}}}`,
		DefaultEntry:    "console.log('hi')",
	}
	zipBytes, err := PackZip(files)
	if err != nil {
		t.Fatal(err)
	}
	out, err := UnpackZip(zipBytes)
	if err != nil {
		t.Fatal(err)
	}
	if out[DefaultEntry] != files[DefaultEntry] {
		t.Fatalf("pet.js mismatch")
	}
}

func TestUnpackZip_stripsSingleRootFolder(t *testing.T) {
	inner := map[string]string{
		ManifestFile: `{"version":1,"type":"sprite","name":"X","assets":{"sprite":{"baseUrl":"assets/","animations":{"idle":{"files":["a.png"],"frameWidth":1,"frameHeight":1,"frames":1}}}}}`,
		DefaultEntry: "// ok",
	}
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, body := range inner {
		f, err := w.Create("我的桌宠.soulpet/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := UnpackZip(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out[ManifestFile]; !ok {
		t.Fatalf("expected manifest at root, keys: %v", keys(out))
	}
	if _, ok := out[DefaultEntry]; !ok {
		t.Fatalf("expected pet.js at root")
	}
}

func keys(m map[string]string) []string {
	k := make([]string, 0, len(m))
	for s := range m {
		k = append(k, s)
	}
	return k
}
