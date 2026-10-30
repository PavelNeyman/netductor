class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.612"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.612/netductor-op-darwin-arm64"
      sha256 "17850f7b63e0ebe9c932353e92b75432e8fdc5855c88c687a660c935c9d96f1e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.612/netductor-op-darwin-amd64"
      sha256 "c0f9364aac3e248025816787b02039991e7d938aaf0f11c7a7b28a04ec44643b"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
