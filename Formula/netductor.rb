class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.136"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.136/netductor-op-darwin-arm64"
      sha256 "252d6fae70ab8dbba5c7ebdcff19f96b63ff9b665ff9ed6dd502da3ea29379fe"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.136/netductor-op-linux-amd64"
      sha256 "5d90675a930e5076451ff51c632c5f3e4b4ee91ad01bbb637f6d6e8d7038f514"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
