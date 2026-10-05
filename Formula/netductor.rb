class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.251"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.251/netductor-op-darwin-arm64"
      sha256 "709e0a8b9240d5ba4f901c0cf5ce86a3e7f40f87bfc93a6e37011f955a9a6ba1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.251/netductor-op-darwin-amd64"
      sha256 "f1cd96d287ce4b342234d1b08cbba81e25dc4820ea9f51d9a76cc8b01bc8106d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.251/netductor-op-linux-amd64"
      sha256 "3d8f1e266a977d972e705f47fc826c2528e7b8e2d15b099d066226d58dd110f6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
