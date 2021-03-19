// +build cgo

package d2xx

/*
#cgo CFLAGS: -I${SRCDIR}/native
#cgo LDFLAGS: -L${SRCDIR}/native/windows_amd64 -lftd2xx64
*/
import "C"
