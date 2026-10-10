class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.320"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.320/netductor-op-darwin-arm64"
      sha256 "382e8a18dd57fc59fb9d457629a5ea3e7e2b7246eb38be4f84ff244ee2446b2b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.320/netductor-op-darwin-amd64"
      sha256 "b7e0b2e78d0e8bed5d412c0caed084bc3fe16bc4f6ead9fa569acb200f88e2d2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.320/netductor-op-linux-amd64"
      sha256 "45673efd30d14341e56a2d73fe46628b2b713605af15df22329682808987276c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
