class GoFastCli < Formula
  desc "Command-line tool to test internet speed using Fast.com with a live TUI"
  homepage "https://github.com/orekasep/go-fast-cli"
  version "1.0.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.1/go-fast-cli_v1.0.1_darwin-arm64.tar.gz"
      sha256 "d7f702b5abd9d3c706f29e306c7d409f74642f5655efa959156ffb54c4ac9842"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.1/go-fast-cli_v1.0.1_darwin-amd64.tar.gz"
      sha256 "43d0593e6ca0e1c56ad852a1e963ce5a33f9e0214bfb1cf3d4ac4166809ecaa1"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.1/go-fast-cli_v1.0.1_linux-arm64.tar.gz"
      sha256 "2561db73ed6328725647500268b9e03b6a91d7341086b635b808ef13ce1e8f3a"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.1/go-fast-cli_v1.0.1_linux-amd64.tar.gz"
      sha256 "100ed9413d5061adf0f7d8f27f3291b4e7018da36f73d7d79fd337246f76b7f9"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
  end

  test do
    assert_match "go-fast-cli", shell_output("#{bin}/go-fast-cli -version")
  end
end
