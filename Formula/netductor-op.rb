class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.784"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.784/netductor-op-darwin-arm64"
      sha256 "45898a284ebd3663dcf2e86eef856562fc934b3a79ff455f63454fc1275c8436"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.784/netductor-op-darwin-amd64"
      sha256 "49fe2568ff5f001e5d115387f07dbdfcfd10f6d64d8ffed808e26a30dbf57eff"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
