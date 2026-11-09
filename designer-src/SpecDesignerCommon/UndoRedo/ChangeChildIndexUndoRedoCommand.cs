using System;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000126 RID: 294
	public class ChangeChildIndexUndoRedoCommand : AbstractUndoRedoCommand
	{
		// Token: 0x06000A4B RID: 2635 RVA: 0x000336AB File Offset: 0x000318AB
		public ChangeChildIndexUndoRedoCommand(XmlElement element, int newIndex)
			: base(element)
		{
			this._oldIndex = element.Index;
			this._newIndex = newIndex;
		}

		// Token: 0x06000A4C RID: 2636 RVA: 0x000336C8 File Offset: 0x000318C8
		public override void Undo()
		{
			XmlElement parent = base.Element.Parent;
			parent.RemoveNode(base.Element);
			parent.AddNodeAt(base.Element, this._oldIndex);
			ComponentHelper.Get(base.Element.Key).AddSelection(base.Element, false);
		}

		// Token: 0x06000A4D RID: 2637 RVA: 0x0003371C File Offset: 0x0003191C
		public override void Execute()
		{
			XmlElement parent = base.Element.Parent;
			parent.RemoveNode(base.Element);
			parent.AddNodeAt(base.Element, this._newIndex);
			ComponentHelper.Get(base.Element.Key).AddSelection(base.Element, false);
		}

		// Token: 0x040003E6 RID: 998
		private int _newIndex;

		// Token: 0x040003E7 RID: 999
		private int _oldIndex;
	}
}
