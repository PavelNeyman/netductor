class Netductor < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.20"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-darwin-arm64"
      sha256 "8d34116b46260b29d9d7a32fb063ca318b727510af0eafc33a38fbf92d73e2de"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-darwin-amd64"
      sha256 "080fe39d36b656dc3474dbcc8b4a620595564cdee1ea630fc5bfdb48d341db07"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-linux-amd64"
      sha256 "7a1fe001fdff11d20a4d17619dc1d7e21aa7ff2068283c811a135421f5c1c9fb"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-linux-arm64"
      sha256 "0a41a576f7148f31fcaebf6d70b4c9d8051639f4172850dab69051c3d4c61531"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
end
