class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.186"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.186/netductor-op-darwin-arm64"
      sha256 "4d3fbe4ef59c249391ae3bbd5d0c16107697617ce07d1bb338aed3c0460a3cf1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.186/netductor-op-darwin-amd64"
      sha256 "947e9a7df8c5738978adab96a18f2e2ee4d12f9c981b1241073421b6ef703ea5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.186/netductor-op-linux-amd64"
      sha256 "02f95a6cfae6f369bdac6b554447ad97093c52d78b3f72f517b11e447ee38f96"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
