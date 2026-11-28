class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.782"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.782/netductor-op-darwin-arm64"
      sha256 "948c372328f3c21e901addeeca2fbaa993c6b96847fd8d6b3f9083f84b7d7d40"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.782/netductor-op-darwin-amd64"
      sha256 "05c50dc6f6ec4e3e1f2882b6f7c40713b3ceda46dce9a10d8878c002640ac8e5"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
