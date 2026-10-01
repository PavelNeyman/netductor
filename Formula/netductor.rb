class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.155"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.155/netductor-op-darwin-arm64"
      sha256 "799c9831e8c36a2096bb757e4882b97c2c74cd22443e5ccd4d3dd7f75be1a31c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.155/netductor-op-darwin-amd64"
      sha256 "ca5e954efae92ba1467d35fab0300ffb17760eb57a886dc665717d9c6acc3da7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.155/netductor-op-linux-amd64"
      sha256 "77c53aeea827a3835d10d08b588c43ee05cd51d978623017a846a2d50aeb0cad"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
