class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.172"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.172/netductor-op-darwin-arm64"
      sha256 "1b5b62966628ee5b96c3414d39f3884507e2ae851b701ee3ced050758bad0fc5"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.172/netductor-op-darwin-amd64"
      sha256 "a8316ccc97e795761ed4299ff2444cfe921271084a615b16df201128256307d2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.172/netductor-op-linux-amd64"
      sha256 "ceed9b4843399bf5fa14a64e9fb5898a35ff3c6aaa9eeeaab195959a835b0b73"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
