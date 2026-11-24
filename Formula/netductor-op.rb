class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.768"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.768/netductor-op-darwin-arm64"
      sha256 "7333be0970c7525d60d90b8dd419b7078acc9c030df6163741c8cafcd45da1e8"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.768/netductor-op-darwin-amd64"
      sha256 "c56c0685a6afe70174cd65b2d48350d454ccc7879d863e53bdf62e55bdd52036"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
