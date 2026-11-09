using System;
using System.Linq;
using System.Windows.Interactivity;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Document;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000013 RID: 19
	public class TextChangedBehavior : Behavior<CodeTextEditor>
	{
		// Token: 0x06000087 RID: 135 RVA: 0x000059F0 File Offset: 0x00003BF0
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.Document.Changed += this.Document_Changed;
		}

		// Token: 0x06000088 RID: 136 RVA: 0x00005A14 File Offset: 0x00003C14
		protected override void OnDetaching()
		{
			base.AssociatedObject.Document.Changed -= this.Document_Changed;
			base.OnDetaching();
		}

		// Token: 0x06000089 RID: 137 RVA: 0x00005A58 File Offset: 0x00003C58
		private void Document_Changed(object sender, DocumentChangeEventArgs e)
		{
			if (base.AssociatedObject.Mode == ContentType.ADP)
			{
				PackageKey programKey = base.AssociatedObject.ProgramKey;
				DocumentLine lineByOffset = base.AssociatedObject.Document.GetLineByOffset(e.Offset);
				SegmentObject segment = base.AssociatedObject.Document.SectionProvider.Find(lineByOffset.LineNumber);
				if (segment != null)
				{
					AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(programKey).AddPoints.Where<AddPointModel>((AddPointModel adp) => adp.ID == segment.ID).ElementAtOrDefault<AddPointModel>(0);
					if (addPointModel != null)
					{
						addPointModel.Status = Status.MODIFY | addPointModel.Status;
					}
				}
			}
		}
	}
}
