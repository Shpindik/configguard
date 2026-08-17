// Package dirscan реализует рекурсивный обход директории с конфигами и
// параллельную проверку найденных файлов через worker pool.
package dirscan

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"configguard/internal/application/scanner"
)

// Result — исход проверки одного файла: либо Report, либо ошибка чтения/парсинга.
type Result struct {
	Path   string
	Report scanner.Report
	Err    error
}

// Scan рекурсивно обходит root, находит файлы поддерживаемых форматов и
// проверяет их параллельно через пул из `workers`. При отмене ctx
// оставшиеся файлы не обрабатываются.
func Scan(ctx context.Context, svc *scanner.Service, root string, workers int) ([]Result, error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	paths, err := collectConfigFiles(root)
	if err != nil {
		return nil, err
	}

	jobs := make(chan string)
	results := make(chan Result)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for path := range jobs {
				report, err := svc.ScanFile(path)
				select {
				case results <- Result{Path: path, Report: report, Err: err}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, p := range paths {
			select {
			case jobs <- p:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	all := make([]Result, 0, len(paths))
	for r := range results {
		all = append(all, r)
	}
	if err := ctx.Err(); err != nil {
		return all, err
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	return all, nil
}

var configExtensions = map[string]struct{}{".json": {}, ".yaml": {}, ".yml": {}}

func collectConfigFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("dirscan: обход %q: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := configExtensions[strings.ToLower(filepath.Ext(path))]; ok {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("dirscan: в %q не найдено файлов %s", root, formatSupportedExts())
	}
	return paths, nil
}

func formatSupportedExts() string {
	exts := make([]string, 0, len(configExtensions))
	for e := range configExtensions {
		exts = append(exts, e)
	}
	sort.Strings(exts)
	return strings.Join(exts, ", ")
}
