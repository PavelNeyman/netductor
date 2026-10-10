class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.324"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.324/netductor-op-darwin-arm64"
      sha256 "0243753f5bac01290565b976c2dcb07269e5fe97544a21ffdd3d9870d4dcc3bf"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.324/netductor-op-darwin-amd64"
      sha256 "499881f0f4b9bfce50905ee81995190dcda5a1ea1854c5543ce70d8623b7ed48"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.324/netductor-op-linux-amd64"
      sha256 "9f9c30095cd9267d0adb18927050742d647a8194a0aee2e1efe325c1cf40698e"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
