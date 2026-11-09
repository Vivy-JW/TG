using System;
using System.Collections.Generic;
using System.Windows;
using System.Windows.Interactivity;
using ICSharpCode.AvalonEdit.Controls;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000031 RID: 49
	public class DiffTrackerBehavior : Behavior<CodeTextEditor>
	{
		// Token: 0x060001E8 RID: 488 RVA: 0x0000F7DC File Offset: 0x0000D9DC
		protected override void OnAttached()
		{
			base.OnAttached();
			this._editor = base.AssociatedObject;
			this._programKey = this._editor.ProgramKey;
			base.AssociatedObject.Loaded += this.AssociatedObject_Loaded;
			this._editor.ModeChanged += this._editor_ModeChanged;
		}

		// Token: 0x060001E9 RID: 489 RVA: 0x0000F83A File Offset: 0x0000DA3A
		protected override void OnDetaching()
		{
			base.OnDetaching();
			this._editor.ModeChanged -= this._editor_ModeChanged;
			if (this._scrollViewer != null)
			{
				this._scrollViewer.ClearBookmark(MarkPurpose.Modified);
			}
		}

		// Token: 0x060001EA RID: 490 RVA: 0x0000F86D File Offset: 0x0000DA6D
		private void _editor_ModeChanged(object sender, EventArgs e)
		{
			this.CreateDiffColorArea();
		}

		// Token: 0x060001EB RID: 491 RVA: 0x0000F878 File Offset: 0x0000DA78
		private void AssociatedObject_Loaded(object sender, RoutedEventArgs e)
		{
			base.AssociatedObject.Loaded -= this.AssociatedObject_Loaded;
			this._scrollViewer = this._editor.Template.FindName("PART_ScrollViewer", base.AssociatedObject) as ScrollViewerEnhanced;
			this.CreateDiffColorArea();
		}

		// Token: 0x060001EC RID: 492 RVA: 0x0000F8C8 File Offset: 0x0000DAC8
		private void CreateDiffColorArea()
		{
			if (this._scrollViewer == null || null == this._programKey || ResourceController.GetInstance().GetProgramInfo(this._programKey) == null)
			{
				return;
			}
			this._scrollViewer.ClearDiffColorArea();
			List<DiffColor> diffColorArea = ResourceController.GetInstance().GetProgramInfo(this._programKey).DiffColorArea;
			foreach (DiffColor diffColor in diffColorArea)
			{
				this._scrollViewer.AddDiffColorArea(diffColor);
			}
			this._scrollViewer.SetDiffTotalLineNumber(ResourceController.GetInstance().GetProgramInfo(this._programKey).DiffTotalLineLumber);
		}

		// Token: 0x040000CD RID: 205
		private ScrollViewerEnhanced _scrollViewer;

		// Token: 0x040000CE RID: 206
		private CodeTextEditor _editor;

		// Token: 0x040000CF RID: 207
		private PackageKey _programKey;
	}
}
