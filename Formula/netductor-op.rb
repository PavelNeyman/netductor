class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.328"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.328/netductor-op-darwin-arm64"
      sha256 "0826a77d1c437ff1f3e27737080340469d9129be76bfd29e994bcd9fca3101ac"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.328/netductor-op-darwin-amd64"
      sha256 "b342ebe248e00ffcb17129b46ba5586c2959a14955ab9591dfe1f7d92fdd466a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.328/netductor-op-linux-amd64"
      sha256 "78d53bcef06c3eba806f9eb36c9a5801c964c034ebe93a0e81b59736747a2189"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
