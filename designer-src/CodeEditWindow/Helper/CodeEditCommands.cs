using System;
using System.Windows.Input;
using SpecDesigner.CodeEditWindow.View;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200001B RID: 27
	public static class CodeEditCommands
	{
		// Token: 0x04000066 RID: 102
		public static RoutedCommand CreateFunction = new RoutedCommand("CreateFunction", typeof(CodeEditCommands));

		// Token: 0x04000067 RID: 103
		public static CustomizedRoutedCommand DeleteFunction = new CustomizedRoutedCommand("DeleteFunction", typeof(CodeEditCommands));

		// Token: 0x04000068 RID: 104
		public static RoutedCommand RefreshStructure = new RoutedCommand("RefreshStructure", typeof(CodeEditCommands));

		// Token: 0x04000069 RID: 105
		public static readonly CustomizedRoutedCommand RefreshScreen = new CustomizedRoutedCommand("RefreshScreen", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.F5)
		});

		// Token: 0x0400006A RID: 106
		public static readonly CustomizedRoutedCommand NextSearchResult = new CustomizedRoutedCommand("NextSearchResult", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.F3)
		});

		// Token: 0x0400006B RID: 107
		public static readonly RoutedCommand Rename = new RoutedCommand("Rename", typeof(CodeTextEditor));

		// Token: 0x0400006C RID: 108
		public static readonly RoutedCommand KeywordUpperCase = new RoutedCommand("KeywordUpperCase", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.B, ModifierKeys.Control)
		});

		// Token: 0x0400006D RID: 109
		public static readonly RoutedCommand FullTextKeywordUpperCase = new RoutedCommand("FullTextKeywordUpperCase", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.B, ModifierKeys.Control | ModifierKeys.Shift)
		});

		// Token: 0x0400006E RID: 110
		public static readonly RoutedCommand BlockComment = new RoutedCommand("Block", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.K, ModifierKeys.Control)
		});

		// Token: 0x0400006F RID: 111
		public static readonly RoutedCommand UnblockComment = new RoutedCommand("UnblockComment", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.K, ModifierKeys.Control | ModifierKeys.Shift)
		});

		// Token: 0x04000070 RID: 112
		public static readonly RoutedCommand CodeCompletion = new RoutedCommand("CodeCompletion", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.Return, ModifierKeys.Control)
		});

		// Token: 0x04000071 RID: 113
		public static readonly RoutedCommand FavoriteCodeCompletion = new RoutedCommand("FavoriteCodeCompletion", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.P, ModifierKeys.Control)
		});

		// Token: 0x04000072 RID: 114
		public static readonly RoutedCommand ColumnCodeCompletion = new RoutedCommand("ColumnCodeCompletion", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.Q, ModifierKeys.Control)
		});

		// Token: 0x04000073 RID: 115
		public static readonly CustomizedRoutedCommand ShowFunction = new CustomizedRoutedCommand("ShowFunction", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.I, ModifierKeys.Control)
		});

		// Token: 0x04000074 RID: 116
		public static readonly CustomizedRoutedCommand FramMark = new CustomizedRoutedCommand("FramMark", typeof(CodeTextEditor));

		// Token: 0x04000075 RID: 117
		public static readonly RoutedCommand GoToLine = new RoutedCommand("GoToLine", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.G, ModifierKeys.Control)
		});

		// Token: 0x04000076 RID: 118
		public static readonly RoutedCommand GoToDiffLine = new RoutedCommand("GoToDiffLine", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.G, ModifierKeys.Control | ModifierKeys.Shift)
		});

		// Token: 0x04000077 RID: 119
		public static readonly CustomizedRoutedCommand InsertCodeSample = new CustomizedRoutedCommand("InsertCodeSample", typeof(CodeTextEditor));

		// Token: 0x04000078 RID: 120
		public static readonly CustomizedRoutedCommand CiteOrNot = new CustomizedRoutedCommand("CiteOrNot", typeof(CodeTextEditor));

		// Token: 0x04000079 RID: 121
		public static readonly CustomizedRoutedCommand CopyAllColumnsToRecord = new CustomizedRoutedCommand("CopyAllColumnsToRecord", typeof(CodeTextEditor));

		// Token: 0x0400007A RID: 122
		public static readonly CustomizedRoutedCommand CopyColumnsName = new CustomizedRoutedCommand("CopyColumnsName", typeof(CodeTextEditor));

		// Token: 0x0400007B RID: 123
		public static readonly CustomizedRoutedCommand ReturnToStandard = new CustomizedRoutedCommand("ReturnToStandard", typeof(CodeTextEditor));

		// Token: 0x0400007C RID: 124
		public static readonly CustomizedRoutedCommand ShowAddPointName = new CustomizedRoutedCommand("ShowAddPointName", typeof(CodeTextEditor));

		// Token: 0x0400007D RID: 125
		public static readonly CustomizedRoutedCommand RefreshDiffResult = new CustomizedRoutedCommand("RefreshScreen", typeof(CodeTextEditor), new InputGestureCollection
		{
			new KeyGesture(Key.D, ModifierKeys.Control)
		});

		// Token: 0x0400007E RID: 126
		public static readonly CustomizedRoutedCommand EnabledFunctionOrder = new CustomizedRoutedCommand("EnabledFunctionOrder", typeof(CodeTextEditor));

		// Token: 0x0400007F RID: 127
		public static readonly CustomizedRoutedCommand DiffSingleContent = new CustomizedRoutedCommand("DiffSingleContent", typeof(CodeTextEditor));

		// Token: 0x04000080 RID: 128
		public static readonly CustomizedRoutedCommand DiffBaseOnStandard = new CustomizedRoutedCommand("DiffBaseOnStandard", typeof(CodeTextEditor));

		// Token: 0x04000081 RID: 129
		public static readonly CustomizedRoutedCommand DiffBlockCopyContent = new CustomizedRoutedCommand("DiffBlockCopyContent", typeof(DiffTextViewer));

		// Token: 0x04000082 RID: 130
		public static readonly CustomizedRoutedCommand DiffLineCopyContent = new CustomizedRoutedCommand("DiffLineCopyContent", typeof(DiffTextViewer));

		// Token: 0x04000083 RID: 131
		public static readonly CustomizedRoutedCommand DiffFunctionSync = new CustomizedRoutedCommand("DiffCopyNewFunction", typeof(DiffTextViewer));

		// Token: 0x04000084 RID: 132
		public static readonly CustomizedRoutedCommand GenerateSqlTemplate = new CustomizedRoutedCommand("GenerateSqlTemplate", typeof(CodeTextEditor));
	}
}
