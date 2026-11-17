class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.757"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.757/netductor-op-darwin-arm64"
      sha256 "0b7237a16a75a9de632d8475fab0e13be4b3a24cfddb0380260c43970921fd0c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.757/netductor-op-darwin-amd64"
      sha256 "727ce46c7a72204785940378cf2d27d24227129072be5a21d46ff96c20d110a6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
