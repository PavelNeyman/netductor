class Netductor < Formula
  desc "Netductor operator"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.126"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.126/netductor-op-darwin-arm64"
      sha256 "99a4ba42d578b947d9119c35046e508f2c1a264414d9ab9a7bf9b3d0ab380e3c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.126/netductor-op-darwin-amd64"
      sha256 "a4067c78444e595078382834d9a0e5b67ab60a0409c1bb3277611ad6ed7731ce"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.126/netductor-op-linux-amd64"
      sha256 "4f5e91d20c6de11ea9490da0b2fa127e189e63d7b905af5c30229c9cccfa60c3"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
