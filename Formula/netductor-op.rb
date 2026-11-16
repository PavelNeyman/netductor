class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.755"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.755/netductor-op-darwin-arm64"
      sha256 "79d19213bb93ff9ccbfe3f93ed90defeb22991e46df3dbe855d38858e296ff8f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.755/netductor-op-darwin-amd64"
      sha256 "a843119f2001e1e1ec82bfc947451163f2f4fe0305fc012b54958175075a7200"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
