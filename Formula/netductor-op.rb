class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.406"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.406/netductor-op-darwin-arm64"
      sha256 "cc930916237f0c5c6a75a98c47e9fecff9b40c435b5c4ab0c311592d0c8c151a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.406/netductor-op-darwin-amd64"
      sha256 "fe723e9bf72933c859abc959b0f69151ac3921d51569f0425beafe0b35eeb1e5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.406/netductor-op-linux-amd64"
      sha256 "23c243f52157a75a7047175a5c0fbe877b6c7354194f51f1477a77ba431ffb67"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
