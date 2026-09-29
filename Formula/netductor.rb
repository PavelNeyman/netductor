class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.111"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.111/netductor-op-darwin-arm64"
      sha256 "bc2aae87d25e554cefc2b9fa496b6ec40cc46cc11557938a9017f7a2e4200d20"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.111/netductor-op-linux-amd64"
      sha256 "e4308ae78a4c9ed9414469b34b55c672b3b66d870799f8a4d9de3b20b060099c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
