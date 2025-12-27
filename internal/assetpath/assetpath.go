// Package assetpath defines the assetpath type
package assetpath

type AssetPath string

func (ap AssetPath) String() string {
	return string(ap)
}
