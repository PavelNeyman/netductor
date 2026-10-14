class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.387"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.387/netductor-op-darwin-arm64"
      sha256 "a7d2c839710c08f7a30517ca2ee45d44ca1f40a29770291deb2253baccb88ae7"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.387/netductor-op-darwin-amd64"
      sha256 "3764a3be0568a1f1ce1ded418da5917430b51b5ee95ff0a9f112bc11e0c5fc6e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.387/netductor-op-linux-amd64"
      sha256 "bb2e2285ed2d64b71ead48bd66ba833f21a8a31103091263c7fcb60e1a93e231"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
