package main

import (
	"fmt"
	"os"

	"github.com/tealeg/xlsx"
)

func SetExcelRowValue(excelFileName string, rowIdx int, rowValues []float64) {
	rowIdx++
	// 检查文件是否存在，如果不存在则创建一个新的文件
	if _, err := os.Stat(excelFileName); os.IsNotExist(err) {
		// 创建一个新的 Excel 文件
		newFile := xlsx.NewFile()
		sheet, _ := newFile.AddSheet("Sheet1")
		for range rowValues {
			sheet.AddRow()
		}
		if err := newFile.Save(excelFileName); err != nil {
			fmt.Printf("Failed to create Excel file: %s\n", err)
			return
		}
	}

	// 打开或创建 Excel 文件
	excelFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		fmt.Printf("Failed to open or create Excel file: %s\n", err)
		return
	}

	// 获取第一个Sheet
	sheet, ok := excelFile.Sheet["Sheet1"] // 使用索引0获取第一个Sheet
	if !ok {
		fmt.Println("Sheet1 does not exist in file")
		return
	}

	// 确保有足够的行，如果不足则添加
	for len(sheet.Rows) <= rowIdx {
		sheet.AddRow()
	}

	// 获取指定的行
	row := sheet.Rows[rowIdx]

	// 确保行中有足够的单元格，如果不足则添加
	for len(row.Cells) < len(rowValues)+1 {
		row.AddCell()
	}

	// 遍历RowValues，将值设置到行的对应单元格中
	for colIdx, value := range rowValues {
		cell := row.Cells[colIdx+1]
		cell.SetFloat(value)
	}

	// 保存Excel文件
	if err = excelFile.Save(excelFileName); err != nil {
		fmt.Printf("Failed to save Excel file: %s\n", err)
		return
	}
	fmt.Printf("Row at index %d has been updated with new values.\n", rowIdx+1)
}
