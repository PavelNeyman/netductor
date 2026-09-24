class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.90"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.90/netductor-darwin-arm64"
      sha256 "d76b9b670f6ac05bb5c44b9703d8db8de6c08d9e553c3e6c122642d49b6b86af"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.90/netductor-darwin-amd64"
      sha256 "d70bab88b174f138e58a70730c40d3817bdd6d8fbcda496291dee3b747666082"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.90/netductor-linux-amd64"
      sha256 "7c330575686563df55cb958c157351283d7e7843d48e73b733b126bb71d66f50"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.90/netductor-linux-arm64"
      sha256 "2b45c31680bd6ddf5a4f60bd3f72ab6ffa4e7f6a232ef2a4913cdf3668614ac0"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
