using System;
using System.Windows.Input;
using SpecDesigner.CodeEditWindow.View;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200000E RID: 14
	public class CodeSpecificationCommands
	{
		// Token: 0x04000035 RID: 53
		public static readonly RoutedCommand AddItemCommand = new RoutedCommand("AddItemCommand", typeof(CodeSpecificationMainWindow));

		// Token: 0x04000036 RID: 54
		public static readonly RoutedCommand DeleteCommand = new RoutedCommand("DeleteCommand", typeof(CodeSpecificationMainWindow));

		// Token: 0x04000037 RID: 55
		public static readonly RoutedCommand CopyCommand = new RoutedCommand("CopyCommand", typeof(CodeSpecificationMainWindow));

		// Token: 0x04000038 RID: 56
		public static readonly RoutedCommand CreateFunctionCommand = new RoutedCommand("CreateFunctionCommand", typeof(CodeSpecificationMainWindow));

		// Token: 0x04000039 RID: 57
		public static readonly RoutedCommand FocusFunctionCommand = new RoutedCommand("FocusFunctionCommand", typeof(CodeSpecificationMainWindow));
	}
}
