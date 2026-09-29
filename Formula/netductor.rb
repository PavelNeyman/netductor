class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.94"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.94/netductor-op-darwin-arm64"
      sha256 "18975fe52f84c0c096e6bf1e45245fe076a13148359c14db53dc011da53e670e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.94/netductor-op-darwin-amd64"
      sha256 "0e6e345d97cf4806ad0830cfce9870961907e5455d1e5a0544f0e0188d08340b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.94/netductor-op-linux-amd64"
      sha256 "dbf5a1dcb7fd1f3277e41e30de35b9caa86815007ee3324714c0b923bd06d97f"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.94/netductor-op-linux-arm64"
      sha256 "650f342e889cca7c181f5527d2a78058c28ad0bd0b9d94bd77d064e26f69043f"
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
