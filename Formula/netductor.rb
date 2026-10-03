class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.205"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.205/netductor-op-darwin-arm64"
      sha256 "c1c0b3846890c42e0eefa9cdc4f30ac0dbf9e463806b9d777880f6ccb882ebdc"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.205/netductor-op-darwin-amd64"
      sha256 "50ca61f60cac8374114587c18d4eebbc776c0a32090fb345e867574eec85d2c7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.205/netductor-op-linux-amd64"
      sha256 "c607c5a5f4ffe730a790bcd7140c6bce8ebafdfd5aec76ed15a5cca9003d1952"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
