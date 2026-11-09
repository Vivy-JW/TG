using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using SpecDesigner.CodeEditWindow.View;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200000D RID: 13
	public class EditorLineValidationRule : ValidationRule
	{
		// Token: 0x0600006F RID: 111 RVA: 0x00005620 File Offset: 0x00003820
		public EditorLineValidationRule(CodeTextEditor editor)
		{
			this.Editor = editor;
			this.Min = 1;
			this.Max = this.Editor.LineCount;
			this.LineOnCaret = editor.Document.GetLineByOffset(editor.CaretOffset).LineNumber;
		}

		// Token: 0x1700000F RID: 15
		// (get) Token: 0x06000070 RID: 112 RVA: 0x0000566E File Offset: 0x0000386E
		// (set) Token: 0x06000071 RID: 113 RVA: 0x00005676 File Offset: 0x00003876
		public int Min { get; private set; }

		// Token: 0x17000010 RID: 16
		// (get) Token: 0x06000072 RID: 114 RVA: 0x0000567F File Offset: 0x0000387F
		// (set) Token: 0x06000073 RID: 115 RVA: 0x00005687 File Offset: 0x00003887
		public int Max { get; private set; }

		// Token: 0x17000011 RID: 17
		// (get) Token: 0x06000074 RID: 116 RVA: 0x00005690 File Offset: 0x00003890
		// (set) Token: 0x06000075 RID: 117 RVA: 0x00005698 File Offset: 0x00003898
		public int LineOnCaret
		{
			get
			{
				return this._lineOnCaret;
			}
			set
			{
				this._lineOnCaret = value;
			}
		}

		// Token: 0x06000076 RID: 118 RVA: 0x000056A4 File Offset: 0x000038A4
		public override ValidationResult Validate(object value, CultureInfo cultureInfo)
		{
			int num = 0;
			try
			{
				if (((string)value).Length > 0)
				{
					num = int.Parse((string)value);
				}
			}
			catch (Exception)
			{
				return new ValidationResult(false, Application.Current.FindResource("Message_IllegalInput") as string);
			}
			if (num < this.Min || num > this.Max)
			{
				return new ValidationResult(false, string.Format(Application.Current.FindResource("Message_InputRange") as string, this.Min, this.Max));
			}
			return new ValidationResult(true, null);
		}

		// Token: 0x04000031 RID: 49
		private CodeTextEditor Editor;

		// Token: 0x04000032 RID: 50
		private int _lineOnCaret;
	}
}
