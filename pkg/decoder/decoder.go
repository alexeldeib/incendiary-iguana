package decoder

import (
	"bufio"
	"bytes"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type yamlDecoder struct {
	reader  *yaml.YAMLReader
	decoder runtime.Decoder
	close   func() error
}

// Modified from https://github.com/kubernetes-sigs/cluster-api. Improved to accomodate custom schemes.
func (d *yamlDecoder) Decode(defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	for {
		doc, err := d.reader.Read()
		if err != nil {
			return nil, nil, err
		}

		//  Skip over empty documents, i.e. a leading `---`
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		return d.decoder.Decode(doc, defaults, into)
	}

}

func (d *yamlDecoder) Close() error {
	return d.close()
}

func NewYAMLDecoder(r io.ReadCloser, scheme *runtime.Scheme) streaming.Decoder {
	return &yamlDecoder{
		reader:  yaml.NewYAMLReader(bufio.NewReader(r)),
		decoder: serializer.NewCodecFactory(scheme).UniversalDeserializer(),
		close:   r.Close,
	}
}
