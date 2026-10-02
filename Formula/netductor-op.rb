class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.182"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.182/netductor-op-darwin-arm64"
      sha256 "dd29e2422e43d4461937b2654ae8244bf9d662423608ea078a4ef0343be782d2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.182/netductor-op-darwin-amd64"
      sha256 "02765f237093cefe4bc71bf6f0a0bef8a05fbead43f428d1b678e938ef15f15a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.182/netductor-op-linux-amd64"
      sha256 "3ef694e047ee611ad4e0ba81ef0c4015e9c2d012576a0c42d3bbcb9d15b3385e"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
