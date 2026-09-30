class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.144"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.144/netductor-op-darwin-arm64"
      sha256 "9ab308ece2e9a8beeff9d6977520877fe85ace401aee2825117767b454520f02"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.144/netductor-op-linux-amd64"
      sha256 "4df00e87594e62d33a32c9faa479034e9fbca5ba90e9d300bb2c886febf90913"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
