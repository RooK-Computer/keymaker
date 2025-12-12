package assets

import _ "embed"

//go:embed ../assets/logo.png
var LogoPNG []byte

//go:embed ../assets/BerkeleyMonoTrial-Regular.otf
var FontTTF []byte
