class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.168"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.168/netductor-op-darwin-arm64"
      sha256 "593f6c6b50262af2c7f4ea5afd0037080c6ca432ed4abae7c610da363b59d01c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.168/netductor-op-darwin-amd64"
      sha256 "058db316fa899ef0d89e649ced19987449cfc88c1e8dcd49fc14254f11fbac7d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.168/netductor-op-linux-amd64"
      sha256 "97fa57e4eb157eda4a030da6166a9462897068cbbc37567389f4ef08defbb581"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
