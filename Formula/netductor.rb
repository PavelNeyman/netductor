class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.165"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.165/netductor-op-darwin-arm64"
      sha256 "54653464eb46d05fc77e53afb41537ba62dcea7f2998e56d20e3714e7b976f2d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.165/netductor-op-darwin-amd64"
      sha256 "8a114799a97a0cd117faec2933bcc7ca88c3d003d15cebe3bff36c671a5c6959"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.165/netductor-op-linux-amd64"
      sha256 "3395590ed7f7d8cf1359080fb225fe355dfbbd704a44bf9ade6d574ad56161fb"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
