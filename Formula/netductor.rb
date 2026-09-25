class Netductor < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.18"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-darwin-arm64"
      sha256 "be2b025be7cbe929c13817a0985c393d2c108b2ea0b83dbdbca537ce48dc9735"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-darwin-amd64"
      sha256 "6e65abe7dfd830c3be146cf98d27782434391871b94760864dfb674db37a5ed5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-linux-amd64"
      sha256 "4603830b5773796baa0717f0594a75569920dde98659fcca9e1fb3023380d0da"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-linux-arm64"
      sha256 "69a77e78064c9a5c5b3014d5b4ed826e45257257d38bd7c9e8836ad600a68c90"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
end
