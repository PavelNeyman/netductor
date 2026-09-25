class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.8"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.8/netductor-op-darwin-arm64"
      sha256 "56c0bb47f37ae5f02950b4809677778ef311402e2201d0ac37bc64a5fb63876e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.8/netductor-op-darwin-amd64"
      sha256 "27252da65dcc026d1fb426dbd1e7fbc02d4528fa17ffa9f4a1b9e3c1b260a895"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.8/netductor-op-linux-amd64"
      sha256 "ced2a022a620be4e74e1acc971b38a65be4c26691e81b655cca18e43e36d9be5"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.8/netductor-op-linux-arm64"
      sha256 "fab256c497e04a923ba756be8678af610d5aec3fe66425e65774f7651a39b724"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
