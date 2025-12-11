package log

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"testing"
)

func TestSlogDefault(t *testing.T) {
	slog.Debug("hello", "count", 3)
	slog.Info("hello", "count", 3)
	slog.Warn("hello", "count", 3)
	slog.Error("hello", "count", 3)
}

func TestSlogText(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	logger.Debug("hello", "count", 3)
	logger.Info("hello", "count", 3, "name", "xxx")
	logger.Warn("hello", "count", 3)
	logger.Error("hello", "count", 3)
}

func TestSlogJson(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	logger.Debug("hello", "count", 3)
	logger.Info("hello", "count", 3, "name", "xxx")
	logger.Warn("hello", "count", 3)
	logger.Error("hello", "count", 3)
}

type TypeCheckStruct struct {
	name string
}

func TestTypeCheck(t *testing.T) {
	var table any = map[string]string{
		"k1": "v1",
	}
	var intX any = 1
	typeCheckStruct := TypeCheckStruct{
		name: "xxx",
	}
	typeCheckStructPdr := &TypeCheckStruct{
		name: "xxx",
	}
	fmt.Println(reflect.TypeOf(table))
	fmt.Println(reflect.TypeOf(intX))
	fmt.Println(reflect.TypeOf(typeCheckStruct))
	fmt.Println(reflect.TypeOf(typeCheckStructPdr))
	switch v := table.(type) {
	case map[any]any:
		fmt.Printf("map, %T\n", v)
	default:
		fmt.Printf("default, %T\n", v)
	}
}
