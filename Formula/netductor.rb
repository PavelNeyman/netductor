class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.12"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.12/netductor-op-darwin-arm64"
      sha256 "cc97f9788a576eada3e34647b1e2377513b5358dd971e3aae48aeaf2c2c07591"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.12/netductor-op-darwin-amd64"
      sha256 "c5f067e9bce53a94cee97e74adb463587da2e558734c0a5ce9dadac9629f1f5b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.12/netductor-op-linux-amd64"
      sha256 "d45b26c5270cc2ae1aa69e6f89ee016c842dae09c52279895f52cf2d2b1cdfb0"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.12/netductor-op-linux-arm64"
      sha256 "b275ffb7dcef5a9ba68302005747e5b57d9511b2ca4a886ef1ed36dcaf019092"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
