package cli

type Flags string

const (
	PackageFlag        Flags = "package"
	PackageFlagDefault Flags = "com.example.newapp"
	OsFlag             Flags = "os"
	OsFlagDefault      Flags = "android,ios,windows.macos"
)
