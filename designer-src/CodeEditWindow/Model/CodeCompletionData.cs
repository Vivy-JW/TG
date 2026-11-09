using System;
using System.Windows.Media;
using ICSharpCode.AvalonEdit.CodeCompletion;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Editing;
using SpecDesigner.Infrastructure.Model;

namespace SpecDesigner.CodeEditWindow.Model
{
	// Token: 0x0200002E RID: 46
	public class CodeCompletionData : ICompletionData
	{
		// Token: 0x17000040 RID: 64
		// (get) Token: 0x060001C6 RID: 454 RVA: 0x0000F329 File Offset: 0x0000D529
		// (set) Token: 0x060001C7 RID: 455 RVA: 0x0000F331 File Offset: 0x0000D531
		public IntellisenseEnum Type { get; private set; }

		// Token: 0x17000041 RID: 65
		// (get) Token: 0x060001C8 RID: 456 RVA: 0x0000F33A File Offset: 0x0000D53A
		// (set) Token: 0x060001C9 RID: 457 RVA: 0x0000F342 File Offset: 0x0000D542
		public string Text { get; private set; }

		// Token: 0x17000042 RID: 66
		// (get) Token: 0x060001CA RID: 458 RVA: 0x0000F34B File Offset: 0x0000D54B
		public object Content
		{
			get
			{
				return this.Text;
			}
		}

		// Token: 0x17000043 RID: 67
		// (get) Token: 0x060001CB RID: 459 RVA: 0x0000F353 File Offset: 0x0000D553
		// (set) Token: 0x060001CC RID: 460 RVA: 0x0000F35B File Offset: 0x0000D55B
		public string DescriptionText { get; set; }

		// Token: 0x17000044 RID: 68
		// (get) Token: 0x060001CD RID: 461 RVA: 0x0000F364 File Offset: 0x0000D564
		public object Description
		{
			get
			{
				return this.DescriptionText;
			}
		}

		// Token: 0x17000045 RID: 69
		// (get) Token: 0x060001CE RID: 462 RVA: 0x0000F36C File Offset: 0x0000D56C
		public double Priority
		{
			get
			{
				return 1.0;
			}
		}

		// Token: 0x17000046 RID: 70
		// (get) Token: 0x060001CF RID: 463 RVA: 0x0000F377 File Offset: 0x0000D577
		public ImageSource Image
		{
			get
			{
				return null;
			}
		}

		// Token: 0x060001D0 RID: 464 RVA: 0x0000F37A File Offset: 0x0000D57A
		public void Complete(TextArea textArea, ISegment completionSegment, EventArgs insertionRequestEventArgs)
		{
			textArea.Document.Replace(completionSegment, this.Text);
		}

		// Token: 0x060001D1 RID: 465 RVA: 0x0000F38E File Offset: 0x0000D58E
		public void Complete(CompletionContext context)
		{
			context.Editor.Document.Replace(context.StartOffset, context.Length, this.Text);
			context.EndOffset = context.StartOffset + this.Text.Length;
		}

		// Token: 0x060001D2 RID: 466 RVA: 0x0000F3CC File Offset: 0x0000D5CC
		public static CodeCompletionData Create(IIntellisenseModel model)
		{
			if (model == null)
			{
				return null;
			}
			return new CodeCompletionData
			{
				Text = model.Name,
				DescriptionText = model.ToString(),
				Type = model.Type
			};
		}

		// Token: 0x060001D3 RID: 467 RVA: 0x0000F40C File Offset: 0x0000D60C
		public static CodeCompletionData Create(string name)
		{
			if (name.Length == 0)
			{
				return null;
			}
			return new CodeCompletionData
			{
				Text = name,
				DescriptionText = name,
				Type = IntellisenseEnum.Self
			};
		}
	}
}
