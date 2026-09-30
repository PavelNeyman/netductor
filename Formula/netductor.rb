class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.120"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.120/netductor-op-darwin-arm64"
      sha256 "c87c8b34353f032e8653d5af6c281783d551eef0074209be180a84cdf8ed522f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.120/netductor-op-darwin-amd64"
      sha256 "29ca6387005e4b586fb21b69449605c7b7634cb0873ba3c2b4ca2ac0b64e7222"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.120/netductor-op-linux-amd64"
      sha256 "c32007cb1da0df66af141549917b7be4bfabd98c628f91d0338b86f4448546bd"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
