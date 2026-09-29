class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.98"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.98/netductor-op-darwin-arm64"
      sha256 "761d2e08451a7242a2ac5c3bc70fef6de30319da3875834ca28512bc4fe4320f"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.98/netductor-op-linux-amd64"
      sha256 "7fd173554c4a37593d43656cac8921c57257d9bf79304d225d172910204cb393"
    end
  end

  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    # No symlink to "netductor" — node binary is VPS-only
  end

  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
