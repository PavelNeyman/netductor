class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.69"
  license "MIT"

  # Prefer: netductor-op update (verifies Release when SHA256SUMS present).
  # Formula SHA: refresh after each release (shasum -a 256 netductor-op-darwin-*).
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-darwin-arm64"
      sha256 :no_check
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-darwin-amd64"
      sha256 :no_check
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-linux-amd64"
      sha256 :no_check
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-linux-arm64"
      sha256 :no_check
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
