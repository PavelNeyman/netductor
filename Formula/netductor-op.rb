class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.394"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.394/netductor-op-darwin-arm64"
      sha256 "8d0e443e082e4bd0d3c7901c8fdc9a7bc8380679505edc062cc28fd91a064876"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.394/netductor-op-darwin-amd64"
      sha256 "55518dc3740a2bacee3f5257490d95c403faae267429c704bacef6677ba90fd1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.394/netductor-op-linux-amd64"
      sha256 "2616ec2c175874eac9a29b0ea1df20ea029ff4240ac4ce6b4e3c6e98e56c54c3"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
