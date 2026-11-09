using System;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000099 RID: 153
	public class ConvertWidgetTypeUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x06000639 RID: 1593 RVA: 0x0001C37C File Offset: 0x0001A57C
		public ConvertWidgetTypeUndoRedoCommand(XmlElement element, ComponentType type)
		{
			if (element == null)
			{
				return;
			}
			this._key = element.Key;
			this._oldElement = element;
			this._newElement = ComponentFactory.ConvertTo(this._oldElement, type);
		}

		// Token: 0x0600063A RID: 1594 RVA: 0x0001C3AD File Offset: 0x0001A5AD
		private void Convert(XmlElement oldElement, XmlElement newElement)
		{
			ComponentHelper.Get(this._key).ClearSelection();
			this.GetSpecificationInfo().ConvertComponentType(oldElement, newElement);
			ComponentHelper.Get(this._key).AddSelection(newElement, false);
		}

		// Token: 0x0600063B RID: 1595 RVA: 0x0001C3DE File Offset: 0x0001A5DE
		public override void Undo()
		{
			this.Convert(this._newElement, this._oldElement);
		}

		// Token: 0x0600063C RID: 1596 RVA: 0x0001C3F4 File Offset: 0x0001A5F4
		public override void Execute()
		{
			GeneralComplexCommand generalComplexCommand = new GeneralComplexCommand(this);
			this.GetUndoRedoManager().StartGroup(generalComplexCommand);
			this.Convert(this._oldElement, this._newElement);
			this.GetUndoRedoManager().EndGroup(generalComplexCommand);
		}

		// Token: 0x0600063D RID: 1597 RVA: 0x0001C432 File Offset: 0x0001A632
		public override void Clear()
		{
		}

		// Token: 0x0600063E RID: 1598 RVA: 0x0001C434 File Offset: 0x0001A634
		private UndoRedoManager GetUndoRedoManager()
		{
			return SettingManager.Get().GetUndoRedoManager(this._key);
		}

		// Token: 0x0600063F RID: 1599 RVA: 0x0001C446 File Offset: 0x0001A646
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._key).SpecificationInfo;
		}

		// Token: 0x04000265 RID: 613
		private XmlElement _newElement;

		// Token: 0x04000266 RID: 614
		private XmlElement _oldElement;

		// Token: 0x04000267 RID: 615
		private PackageKey _key;
	}
}
