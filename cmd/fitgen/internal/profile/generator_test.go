package profile

import (
	"bytes"
	"flag"
	"hash/fnv"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"
)

const (
	testdata      = "testdata"
	fileExt       = ".xlsx"
	goldenSuffix  = ".golden"
	currentSuffix = ".current"
)

var update = flag.Bool("update", false, "update .golden output files")

func init() { flag.Parse() }

var currentSDK = sdks[0]

var defGenOpts = []GeneratorOption{
	WithGenerationTimestamp(false),
}

func relPath(sdkVersion string) string {
	return filepath.Join(testdata, sdkVersion+fileExt)
}

func (p *Profile) WriteTo(w io.Writer) (int64, error) {
	var err error
	var n int
	write := func(buf []byte) {
		if err != nil {
			return
		}
		n, err = w.Write(buf)
	}
	write([]byte("// TYPES\n"))
	write(p.TypesSource)
	write([]byte("// MESSAGES\n"))
	write(p.MessagesSource)
	write([]byte("// PROFILE\n"))
	write(p.ProfileSource)
	write([]byte("// STRINGER TYPE INPUT\n"))
	write([]byte(p.StringerInput))
	write([]byte("\n// MESSAGE NUMS WITHOUT MESSAGE\n"))
	for _, mn := range p.MesgNumsWithoutMessage {
		write([]byte(mn))
		write([]byte{'\n'})
	}
	return int64(n), err
}

func (p *Profile) WriteToFile(path string) error {
	buf := new(bytes.Buffer)
	_, err := p.WriteTo(buf)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, buf.Bytes(), 0644)
}

func (p *Profile) FnvHash() uint32 {
	h := fnv.New32a()
	_, _ = p.WriteTo(h)
	return h.Sum32()
}

type sdk struct {
	version    string
	goldenHash uint32
}

var sdks = []sdk{
	{"16.20", 341779287},
}

func TestGenerator(t *testing.T) {
	for _, sdk := range sdks {
		t.Run(sdk.version, func(t *testing.T) {
			if sdk == currentSDK && testing.Short() {
				t.Skip("skipping test in short mode")
			}
			path := relPath(sdk.version)
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			g, err := NewGenerator(path, data, defGenOpts...)
			if err != nil {
				t.Fatal(err)
			}
			profile, err := g.GenerateProfile()
			if err != nil {
				t.Fatal(err)
			}
			gotHash := profile.FnvHash()
			if gotHash == sdk.goldenHash {
				return
			}
			t.Errorf("profile fingerprint differs: got: %d, want %d", gotHash, sdk.goldenHash)
			if !*update {
				path = path + currentSuffix
			} else {
				path = path + goldenSuffix
			}
			err = profile.WriteToFile(path)
			if err != nil {
				t.Fatalf("error writing output: %v", err)
			}
			if !*update {
				t.Logf("current output written to: %s", path)
			} else {
				t.Logf("%q has been updated", path)
				t.Logf("new fingerprint is: %d", gotHash)
			}
		})
	}
}

var profileSink *Profile

func BenchmarkGenerator(b *testing.B) {
	for _, sdk := range sdks {
		b.Run(sdk.version, func(b *testing.B) {
			path := relPath(sdk.version)
			data, err := ioutil.ReadFile(path)
			if err != nil {
				b.Fatalf("error reading profile workbook: %v", err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				g, err := NewGenerator(path, data, defGenOpts...)
				if err != nil {
					b.Fatal(err)
				}
				profileSink, err = g.GenerateProfile()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
