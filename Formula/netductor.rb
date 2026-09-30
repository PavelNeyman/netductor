class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.139"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.139/netductor-op-darwin-arm64"
      sha256 "871f5fa74a515988063ece55661b7a4d99f8a6f12c91eeee239c9bc838e2e482"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.139/netductor-op-linux-amd64"
      sha256 "54ac883a818d9a197d4d444efb41b5aac21a9873383332bf60f576510df8b19d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
