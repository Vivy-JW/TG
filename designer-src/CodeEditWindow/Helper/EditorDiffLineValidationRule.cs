using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using SpecDesigner.CodeEditWindow.View;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000037 RID: 55
	public class EditorDiffLineValidationRule : ValidationRule
	{
		// Token: 0x06000231 RID: 561 RVA: 0x000111B8 File Offset: 0x0000F3B8
		public EditorDiffLineValidationRule(CodeTextEditor editor)
		{
			this.Editor = editor;
			this.Min = 1;
			this.Max = this.Editor.Text.Replace("\a\r\n", "").Replace("\a\n", "").Split(new char[] { '\n' })
				.Length;
			this.RealLineOnCaret = ((editor.Document.GetLineByOffset(editor.CaretOffset).LineNumber > this.Max) ? this.Max : editor.Document.GetLineByOffset(editor.CaretOffset).LineNumber);
		}

		// Token: 0x17000052 RID: 82
		// (get) Token: 0x06000232 RID: 562 RVA: 0x0001125E File Offset: 0x0000F45E
		// (set) Token: 0x06000233 RID: 563 RVA: 0x00011266 File Offset: 0x0000F466
		public int Min { get; private set; }

		// Token: 0x17000053 RID: 83
		// (get) Token: 0x06000234 RID: 564 RVA: 0x0001126F File Offset: 0x0000F46F
		// (set) Token: 0x06000235 RID: 565 RVA: 0x00011277 File Offset: 0x0000F477
		public int Max { get; private set; }

		// Token: 0x17000054 RID: 84
		// (get) Token: 0x06000236 RID: 566 RVA: 0x00011280 File Offset: 0x0000F480
		// (set) Token: 0x06000237 RID: 567 RVA: 0x00011288 File Offset: 0x0000F488
		public int RealLineOnCaret
		{
			get
			{
				return this._realLineOnCaret;
			}
			set
			{
				this._realLineOnCaret = value;
			}
		}

		// Token: 0x06000238 RID: 568 RVA: 0x00011294 File Offset: 0x0000F494
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

		// Token: 0x040000F9 RID: 249
		private CodeTextEditor Editor;

		// Token: 0x040000FA RID: 250
		private int _realLineOnCaret;
	}
}
