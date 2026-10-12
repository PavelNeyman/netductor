class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.349"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.349/netductor-op-darwin-arm64"
      sha256 "fc944f2140e00fd67b86387ad8f253b143c5ae0fe9641939be99182cf8f9db9b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.349/netductor-op-darwin-amd64"
      sha256 "9cc34403e50c3ab4699b0105ea98b050690f302c04ba3177149a30710edc3da8"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.349/netductor-op-linux-amd64"
      sha256 "f64eb2a3cddb226e69e4f16663758ebc55b20fc20933eeb7ad0ce0f86dbc426f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
