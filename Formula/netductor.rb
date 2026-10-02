class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.171"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.171/netductor-op-darwin-arm64"
      sha256 "d239b1079c8922bd00e5441cc1c07e80a529f2deae176b897ff5360062d41fce"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.171/netductor-op-darwin-amd64"
      sha256 "50afc093e941a93e8c3898fb029e23e0fdf6595442925dd99c0edf80e11870b1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.171/netductor-op-linux-amd64"
      sha256 "b845170f1f2775b5b5917c40c06d9b3418527c4f0ec9f117b6676295454bddb1"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
